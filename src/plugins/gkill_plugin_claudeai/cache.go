package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	_ "modernc.org/sqlite"
)

// pluginCache は conversations.json から組み立てたメッセージを SQLite3 にキャッシュする。
// gkillのキャッシュディレクトリ配下(sdk.CacheDBPath 参照)に保存する。
//
// ロックを2つに分けているのが要点。
//   - mu は db の遅延初期化だけを守る
//   - buildMu は構築どうしだけを直列化する
//   - 読み取り(GetMessages/GetMsgByID/GetConvForMsg/GetStats)はどちらも取らない
//
// 以前は GetMessages が単一ロック下で rebuild を同期実行しており、エクスポートが
// 大きいと gkill のデッドライン(IsAlive 5秒 / 呼び出し30秒)を超えて kill され、
// ロールバック→進捗ゼロ→次の find_kyous でまた最初から、の無限ループに陥っていた。
// 同梱の gkill_plugin_codex / gkill_plugin_claudecode の常駐ビルダ方式へ揃える。
type pluginCache struct {
	mu      sync.Mutex
	buildMu sync.Mutex
	db      *sql.DB
}

var globalCache = &pluginCache{}

// cachedMessage はmsg_cacheテーブルの1行。
type cachedMessage struct {
	MsgID           string
	ConvID          string
	Sender          string
	Text            string
	RelatedTimeUnix int64
	CreateTimeUnix  int64
	UpdateTimeUnix  int64
}

// cacheStats は設定画面に出す統計。ファイルは1バイトも開かずに作る。
type cacheStats struct {
	FileCount     int
	ConvCount     int
	MessageCount  int
	LastScanUnix  int64
	LastScanError string
	BuildState    string
	BuildError    string
	BuildTotal    int
	BuildDone     int
}

// cacheSchemaVersion はキャッシュのスキーマ版。
// 構造やジャーナルモードを変えたら上げること。古いDBは作り直される(キャッシュなので捨ててよい)。
// v2: 自前WAL化 + gen 列導入(バッチcommitの世代管理)で作り直し。
const cacheSchemaVersion = "2"

// openDB はキャッシュDBを開く。何度呼んでも1回しか開かない。
func (c *pluginCache) openDB(pluginDir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.db != nil {
		return nil
	}

	dbPath := sdk.CacheDBPath(pluginDir)

	// sqlite3impl.GetSQLiteDBConnection は使わない。
	// あれは journal_mode を DELETE に固定するので、バックグラウンドで書いている間
	// 読み手が busy_timeout まで待たされる。派生キャッシュは自前でWALを開く。
	db, err := sql.Open("sqlite", "file:"+dbPath+
		"?_txlock=immediate"+
		"&_pragma=busy_timeout(6000)"+
		"&_pragma=journal_mode(WAL)"+
		"&_pragma=synchronous(NORMAL)"+
		"&_pragma=cache_size(-16000)"+
		"&_pragma=temp_store(MEMORY)")
	if err != nil {
		return fmt.Errorf("error at open cache db %s: %w", dbPath, err)
	}
	if err := initSchema(db); err != nil {
		_ = db.Close()
		return err
	}
	c.db = db
	return nil
}

// conn は開いてあるDBを返す。openDB を通っていれば非nil。
func (c *pluginCache) conn() *sql.DB {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.db
}

// initSchema はテーブルを作成する。
// 記録している版が違えば、作り直しのため既存のテーブルを落とす。
func initSchema(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS cache_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("error at init cache_meta: %w", err)
	}

	version := ""
	_ = db.QueryRow(`SELECT value FROM cache_meta WHERE key = 'schema_version'`).Scan(&version)
	if version != cacheSchemaVersion {
		for _, statement := range []string{
			`DROP TABLE IF EXISTS conv_cache`,
			`DROP TABLE IF EXISTS msg_cache`,
			`DELETE FROM cache_meta`,
		} {
			if _, err := db.Exec(statement); err != nil {
				return fmt.Errorf("error at drop old schema: %w", err)
			}
		}
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS conv_cache (
  conv_id          TEXT PRIMARY KEY,
  title            TEXT NOT NULL,
  create_time_unix INTEGER NOT NULL,
  gen              INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS msg_cache (
  msg_id            TEXT PRIMARY KEY,
  conv_id           TEXT NOT NULL,
  sender            TEXT NOT NULL,
  text              TEXT NOT NULL,
  related_time_unix INTEGER NOT NULL,
  create_time_unix  INTEGER NOT NULL,
  update_time_unix  INTEGER NOT NULL,
  gen               INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE INDEX IF NOT EXISTS idx_msg_conv ON msg_cache(conv_id)`,
		`CREATE INDEX IF NOT EXISTS idx_msg_time ON msg_cache(related_time_unix)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("error at init schema: %w", err)
		}
	}

	if _, err := db.Exec(
		`INSERT OR REPLACE INTO cache_meta(key, value) VALUES('schema_version', ?)`,
		cacheSchemaVersion,
	); err != nil {
		return fmt.Errorf("error at record schema version: %w", err)
	}
	return nil
}

func (c *pluginCache) setMeta(key, value string) {
	db := c.conn()
	if db == nil {
		return
	}
	_, _ = db.Exec(`INSERT OR REPLACE INTO cache_meta(key, value) VALUES(?, ?)`, key, value)
}

func (c *pluginCache) getMeta(key string) string {
	db := c.conn()
	if db == nil {
		return ""
	}
	value := ""
	_ = db.QueryRow(`SELECT value FROM cache_meta WHERE key = ?`, key).Scan(&value)
	return value
}

// GetMessages はFindKyous用に全メッセージを返す。
//
// 読み取りは buildMu を取らない。構築中でも「そこまで取り込めたぶん」を返す。
// これが仕様どおりの挙動で、初回呼び出しが空になるのもそのため。
func (c *pluginCache) GetMessages(pluginDir string) ([]cachedMessage, error) {
	if err := c.openDB(pluginDir); err != nil {
		return nil, err
	}
	db := c.conn()
	if db == nil {
		return nil, fmt.Errorf("cache db is not opened")
	}
	rows, err := db.Query(`SELECT msg_id, conv_id, sender, text, related_time_unix, create_time_unix, update_time_unix
		FROM msg_cache ORDER BY related_time_unix DESC`)
	if err != nil {
		return nil, fmt.Errorf("error at query messages: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var msgs []cachedMessage
	for rows.Next() {
		var m cachedMessage
		if err := rows.Scan(&m.MsgID, &m.ConvID, &m.Sender, &m.Text, &m.RelatedTimeUnix, &m.CreateTimeUnix, &m.UpdateTimeUnix); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// GetMsgByID はGetContentHTML用に、msgIDに対応するメッセージ1件と会話タイトルを返す。
func (c *pluginCache) GetMsgByID(pluginDir string, msgID string) (convTitle string, msg cachedMessage, err error) {
	if err = c.openDB(pluginDir); err != nil {
		return
	}
	db := c.conn()
	if db == nil {
		err = fmt.Errorf("cache db is not opened")
		return
	}
	row := db.QueryRow(`
		SELECT m.msg_id, m.conv_id, m.sender, m.text,
		       m.related_time_unix, m.create_time_unix, m.update_time_unix,
		       COALESCE(c.title, '')
		FROM msg_cache m
		LEFT JOIN conv_cache c ON m.conv_id = c.conv_id
		WHERE m.msg_id = ?`, msgID)
	if e := row.Scan(&msg.MsgID, &msg.ConvID, &msg.Sender, &msg.Text,
		&msg.RelatedTimeUnix, &msg.CreateTimeUnix, &msg.UpdateTimeUnix, &convTitle); e != nil {
		err = fmt.Errorf("msg not found: %s", msgID)
	}
	return
}

// GetConvForMsg はGetContentHTML用に、msgIDが属する会話の全メッセージと会話タイトルを返す。
func (c *pluginCache) GetConvForMsg(pluginDir string, msgID string) (convTitle string, msgs []cachedMessage, err error) {
	if err = c.openDB(pluginDir); err != nil {
		return
	}
	db := c.conn()
	if db == nil {
		err = fmt.Errorf("cache db is not opened")
		return
	}
	var convID string
	if e := db.QueryRow(`SELECT conv_id FROM msg_cache WHERE msg_id = ?`, msgID).Scan(&convID); e != nil {
		err = fmt.Errorf("msg not found: %s", msgID)
		return
	}
	if e := db.QueryRow(`SELECT title FROM conv_cache WHERE conv_id = ?`, convID).Scan(&convTitle); e != nil {
		convTitle = ""
	}
	rows, e := db.Query(`SELECT msg_id, conv_id, sender, text, related_time_unix, create_time_unix, update_time_unix
		FROM msg_cache WHERE conv_id = ? ORDER BY related_time_unix ASC`, convID)
	if e != nil {
		err = fmt.Errorf("error at query conv messages: %w", e)
		return
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var m cachedMessage
		if e := rows.Scan(&m.MsgID, &m.ConvID, &m.Sender, &m.Text, &m.RelatedTimeUnix, &m.CreateTimeUnix, &m.UpdateTimeUnix); e != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	err = rows.Err()
	return
}

// GetStats は設定画面用の統計を返す。
//
// GetConfigHTML から呼ぶので、ここで走査してはいけない。
// 5秒の IsAlive を超えるとプロセスが殺される。
func (c *pluginCache) GetStats(pluginDir string) cacheStats {
	stats := cacheStats{}
	if err := c.openDB(pluginDir); err != nil {
		stats.LastScanError = err.Error()
		return stats
	}
	db := c.conn()
	if db == nil {
		stats.LastScanError = "cache db is not opened"
		return stats
	}

	_ = db.QueryRow(`SELECT COUNT(*) FROM conv_cache`).Scan(&stats.ConvCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM msg_cache`).Scan(&stats.MessageCount)

	stats.FileCount = atoiOrZero(c.getMeta("file_count"))
	stats.BuildState = c.getMeta("build_state")
	stats.BuildError = c.getMeta("build_error")
	stats.BuildTotal = atoiOrZero(c.getMeta("build_total_convs"))
	stats.BuildDone = atoiOrZero(c.getMeta("build_done_convs"))
	if unix := atoiOrZero(c.getMeta("last_scan_unix")); unix > 0 {
		stats.LastScanUnix = int64(unix)
	}
	if stats.BuildError != "" {
		stats.LastScanError = stats.BuildError
	}
	return stats
}

// buildBatchConvs は1トランザクションで取り込む会話数。
// 途中で殺されても、このぶんだけやり直せばよい。テストが小さくできるよう var。
var buildBatchConvs = 64

// ingestConvHook はテストが取り込みに割り込むためのフック。本番では nil。
// エラーを返すとそのバッチが失敗し、先行してコミット済みのバッチはそのまま残る。
var ingestConvHook func(convID string) error

// build はソースを走査し、変化があれば全会話をバッチで作り直す。
// ビルダ以外から呼ばないこと。
//
// Claude.ai のエクスポートは conversations.json が丸ごと入れ替わる形なので、
// 差分は「署名(path:mtime:size)が変わったら全会話を読み直す」。ただし取り込みは
// 会話をバッチに分けて1バッチ=1トランザクションでコミットするので、途中で殺されても
// 先行バッチの行は残り、進捗ゼロには戻らない。世代(gen)で古い版を印し、最後にまとめて掃除する。
func (c *pluginCache) build(pluginDir string, src expandedSource) error {
	c.buildMu.Lock()
	defer c.buildMu.Unlock()

	if err := c.openDB(pluginDir); err != nil {
		return err
	}

	c.setMeta("build_state", "scanning")
	c.setMeta("build_error", "")

	files := findConversationFiles(src)
	signature := sourceSignature(files)
	if signature == "" {
		// ソースが見つからない。既存キャッシュは残したままエラーを返す(取得は非致命)。
		c.setMeta("build_state", "idle")
		return fmt.Errorf("conversations.json が見つかりません")
	}
	c.setMeta("file_count", strconv.Itoa(len(files)))

	stored := c.getMeta("source_signature")
	msgCount := 0
	_ = c.conn().QueryRow(`SELECT COUNT(*) FROM msg_cache`).Scan(&msgCount)
	if signature == stored && msgCount > 0 {
		c.setMeta("build_state", "idle")
		c.setMeta("last_scan_unix", strconv.FormatInt(time.Now().Unix(), 10))
		return nil
	}

	convs, err := loadConversations(src)
	if err != nil {
		return err
	}

	// 新しい世代番号。finalize が成功するまで build_gen メタは書かないので、
	// 途中で殺されても次回は同じ gen を再利用する(再 upsert は冪等)。
	gen := atoiOrZero(c.getMeta("build_gen")) + 1

	c.setMeta("build_total_convs", strconv.Itoa(len(convs)))
	c.setMeta("build_done_convs", "0")
	if len(convs) != 0 {
		c.setMeta("build_state", "ingesting")
	}

	batchSize := buildBatchConvs
	if batchSize <= 0 {
		batchSize = 1
	}
	done := 0
	for start := 0; start < len(convs); start += batchSize {
		end := min(start+batchSize, len(convs))
		if err := c.ingestConvs(convs[start:end], gen); err != nil {
			return err
		}
		done += end - start
		c.setMeta("build_done_convs", strconv.Itoa(done))
	}

	if err := c.finalizeBuild(gen, signature); err != nil {
		return err
	}

	c.setMeta("build_state", "idle")
	return nil
}

// ingestConvs は指定した会話群を1トランザクションで upsert する。gen で今回の版を印す。
func (c *pluginCache) ingestConvs(convs []claudeConversation, gen int) error {
	db := c.conn()
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin ingest tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for i := range convs {
		conv := &convs[i]
		if conv.UUID == "" {
			continue
		}
		if ingestConvHook != nil {
			if err := ingestConvHook(conv.UUID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO conv_cache(conv_id, title, create_time_unix, gen) VALUES(?,?,?,?)`,
			conv.UUID, conv.Name, conv.CreatedAt.Unix(), gen); err != nil {
			return fmt.Errorf("error at insert conv %s: %w", conv.UUID, err)
		}
		for _, msg := range conv.ChatMessages {
			if msg.UUID == "" || msg.Text == "" {
				continue
			}
			createTimeUnix := msg.CreatedAt.Unix()
			updateTimeUnix := msg.UpdatedAt.Unix()
			if _, err := tx.Exec(
				`INSERT OR REPLACE INTO msg_cache(msg_id, conv_id, sender, text, related_time_unix, create_time_unix, update_time_unix, gen) VALUES(?,?,?,?,?,?,?,?)`,
				msg.UUID, conv.UUID, msg.Sender, msg.Text, createTimeUnix, createTimeUnix, updateTimeUnix, gen); err != nil {
				return fmt.Errorf("error at insert msg %s: %w", msg.UUID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit ingest tx: %w", err)
	}
	committed = true
	return nil
}

// finalizeBuild は今回の gen で更新されなかった古い行を落とし、署名と gen を確定する。
// この最終トランザクションが成功して初めて「作り直し完了」になる。
func (c *pluginCache) finalizeBuild(gen int, signature string) error {
	db := c.conn()
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin finalize tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.Exec(`DELETE FROM conv_cache WHERE gen != ?`, gen); err != nil {
		return fmt.Errorf("error at delete stale convs: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM msg_cache WHERE gen != ?`, gen); err != nil {
		return fmt.Errorf("error at delete stale msgs: %w", err)
	}
	for key, value := range map[string]string{
		"build_gen":        strconv.Itoa(gen),
		"source_signature": signature,
		"last_scan_unix":   strconv.FormatInt(time.Now().Unix(), 10),
	} {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO cache_meta(key, value) VALUES(?, ?)`, key, value); err != nil {
			return fmt.Errorf("error at write meta %s: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit finalize tx: %w", err)
	}
	committed = true
	return nil
}

// atoiOrZero は数値でない/空の meta 値を0にする。
func atoiOrZero(s string) int {
	value, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return value
}

// unixToTimeFromCache はUnixタイムスタンプをtime.Timeに変換する。
func unixToTimeFromCache(unix int64) time.Time {
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0)
}
