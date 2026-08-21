package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	_ "modernc.org/sqlite"
)

// pluginCache はトランスクリプトから組み立てた発言をSQLite3にキャッシュする。
// gkillのキャッシュディレクトリ配下(sdk.CacheDBPath 参照)に保存し、
// ファイル単位のmtime/サイズで差分更新する。
// ソースは146MB規模になるため、毎回全部を読み直さないことが重要。
//
// ロックを2つに分けているのが要点。
//   - mu は db の遅延初期化だけを守る
//   - buildMu は構築どうしだけを直列化する
//   - 読み取り(GetMessages/GetMessage/GetStats)はどちらも取らない
//
// 兼用すると初回構築のあいだ find_kyous が全部詰まり、
// gkill のデッドライン(IsAlive 5秒 / 呼び出し30秒)でプロセスが殺され続ける。
// SQLite は WAL で開いてあり *sql.DB 自体も並行安全なので、
// 構築中の読み取りは「そこまで取り込めたぶん」を返せばよい。
type pluginCache struct {
	mu      sync.Mutex
	buildMu sync.Mutex
	db      *sql.DB
}

var globalCache = &pluginCache{}

// messageSummary はFindKyous用の軽量な行。body_jsonは読まない。
type messageSummary struct {
	MessageID       string
	Role            string
	SessionID       string
	SessionTitle    string
	Project         string
	Branch          string
	SearchText      string
	RelatedTimeUnix int64
	UpdateTimeUnix  int64
}

// cacheStats は設定画面に出す統計。ファイルは1バイトも開かずに作る。
type cacheStats struct {
	FileCount     int
	MessageCount  int
	LastScanUnix  int64
	LastScanError string
	BuildState    string
	BuildError    string
	BuildTotal    int
	BuildDone     int
}

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

// cacheSchemaVersion はキャッシュのスキーマ版。
// 構造やジャーナルモードを変えたら上げること。古いDBは作り直される(キャッシュなので捨ててよい)。
const cacheSchemaVersion = "3"

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
			`DROP TABLE IF EXISTS message_cache`,
			`DROP TABLE IF EXISTS turn_cache`,
			`DROP TABLE IF EXISTS file_cache`,
			`DELETE FROM cache_meta`,
		} {
			if _, err := db.Exec(statement); err != nil {
				return fmt.Errorf("error at drop old schema: %w", err)
			}
		}
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS file_cache (
  path       TEXT PRIMARY KEY,
  mtime_unix INTEGER NOT NULL,
  size       INTEGER NOT NULL,
  kind       TEXT NOT NULL,
  session_id TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS message_cache (
  message_id        TEXT PRIMARY KEY,
  role              TEXT NOT NULL,
  source_path       TEXT NOT NULL,
  session_id        TEXT NOT NULL,
  session_title     TEXT NOT NULL,
  project           TEXT NOT NULL,
  branch            TEXT NOT NULL,
  message_text      TEXT NOT NULL,
  search_text       TEXT NOT NULL,
  body_json         TEXT NOT NULL,
  related_time_unix INTEGER NOT NULL,
  update_time_unix  INTEGER NOT NULL
)`,
		`CREATE INDEX IF NOT EXISTS idx_msg_time    ON message_cache(related_time_unix)`,
		`CREATE INDEX IF NOT EXISTS idx_msg_session ON message_cache(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_msg_src     ON message_cache(source_path)`,
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

// GetMessages はFindKyous用に全発言の要約を返す。
//
// 読み取りは buildMu を取らない。構築中でも「そこまで取り込めたぶん」を返す。
// これが仕様どおりの挙動で、初回呼び出しが空になるのもそのため。
func (c *pluginCache) GetMessages(pluginDir string) ([]messageSummary, error) {
	if err := c.openDB(pluginDir); err != nil {
		return nil, err
	}
	db := c.conn()
	if db == nil {
		return nil, fmt.Errorf("cache db is not opened")
	}

	rows, err := db.Query(`
		SELECT message_id, role, session_id, session_title, project, branch, search_text,
		       related_time_unix, update_time_unix
		FROM message_cache ORDER BY related_time_unix DESC`)
	if err != nil {
		return nil, fmt.Errorf("error at query messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var messages []messageSummary
	for rows.Next() {
		var t messageSummary
		if err := rows.Scan(&t.MessageID, &t.Role, &t.SessionID, &t.SessionTitle, &t.Project, &t.Branch,
			&t.SearchText, &t.RelatedTimeUnix, &t.UpdateTimeUnix); err != nil {
			continue
		}
		messages = append(messages, t)
	}
	return messages, rows.Err()
}

// GetMessage はGetContentHTML用に、発言1件を本文込みで返す。
func (c *pluginCache) GetMessage(pluginDir string, messageID string) (message, error) {
	var t message
	if err := c.openDB(pluginDir); err != nil {
		return t, err
	}
	db := c.conn()
	if db == nil {
		return t, fmt.Errorf("cache db is not opened")
	}

	var bodyJSON string
	row := db.QueryRow(`SELECT body_json FROM message_cache WHERE message_id = ?`, messageID)
	if err := row.Scan(&bodyJSON); err != nil {
		return t, fmt.Errorf("message not found: %s", messageID)
	}
	if err := json.Unmarshal([]byte(bodyJSON), &t); err != nil {
		return t, fmt.Errorf("error at parse cached message %s: %w", messageID, err)
	}
	return t, nil
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

	_ = db.QueryRow(`SELECT COUNT(*) FROM file_cache WHERE kind != ?`, kindOther).Scan(&stats.FileCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM message_cache`).Scan(&stats.MessageCount)

	stats.BuildState = c.getMeta("build_state")
	stats.BuildError = c.getMeta("build_error")
	stats.BuildTotal = atoiOrZero(c.getMeta("build_total_sessions"))
	stats.BuildDone = atoiOrZero(c.getMeta("build_done_sessions"))
	if unix := atoiOrZero(c.getMeta("last_scan_unix")); unix > 0 {
		stats.LastScanUnix = int64(unix)
	}
	// 構築時のエラーは従来の「スキャン時のエラー」欄にも出す(既存UIの互換)。
	if stats.BuildError != "" {
		stats.LastScanError = stats.BuildError
	}
	return stats
}

// buildBatchSessions は1トランザクションで取り込むセッション数。
// 途中で殺されても、このぶんだけやり直せばよい。テストが小さくできるよう var。
var buildBatchSessions = 16

// ingestSessionHook はテストが取り込みに割り込むためのフック。本番では nil。
// エラーを返すとそのバッチが失敗し、先行してコミット済みのバッチはそのまま残る。
var ingestSessionHook func(sessionID string) error

// build はソースを走査し、変化のあったセッションだけキャッシュを作り直す。
// ビルダ以外から呼ばないこと。
func (c *pluginCache) build(pluginDir string, src expandedSource) error {
	// 構築どうしだけを直列化する。読み取りは待たせない
	c.buildMu.Lock()
	defer c.buildMu.Unlock()

	if err := c.openDB(pluginDir); err != nil {
		return err
	}

	c.setMeta("build_state", "scanning")
	c.setMeta("build_error", "")

	known, err := c.loadFileCache()
	if err != nil {
		return err
	}

	files, scanErr := scanSources(src, known)

	// エージェントIDからセッションIDを引けるようにする(metaファイルは中にセッションIDを持たないため)
	sessionIDOfAgent := map[string]string{}
	for _, f := range files {
		if f.Kind == kindSubAgent {
			sessionIDOfAgent[agentIDFromFileName(f.Path)] = f.SessionID
		}
	}
	sessionIDOfFile := func(f scannedFile) string {
		if f.Kind == kindMeta {
			return sessionIDOfAgent[agentIDFromFileName(f.Path)]
		}
		return f.SessionID
	}

	// 変化のあったファイルからセッションを洗い出す
	dirtySessions := map[string]bool{}
	current := map[string]scannedFile{}
	var changedFiles []scannedFile
	for _, f := range files {
		current[f.Path] = f
		prev, ok := known[f.Path]
		if ok && prev.MtimeUnix == f.MtimeUnix && prev.Size == f.Size {
			continue
		}
		changedFiles = append(changedFiles, f)
		if sid := sessionIDOfFile(f); sid != "" {
			dirtySessions[sid] = true
		}
	}
	var removedPaths []string
	for path, prev := range known {
		if _, ok := current[path]; ok {
			continue
		}
		removedPaths = append(removedPaths, path)
		if prev.SessionID != "" {
			dirtySessions[prev.SessionID] = true
		}
	}

	if len(dirtySessions) == 0 && len(removedPaths) == 0 && len(changedFiles) == 0 {
		c.setMeta("build_state", "idle")
		c.setMeta("last_scan_unix", strconv.FormatInt(time.Now().Unix(), 10))
		return scanErr
	}

	// セッションごとに、メイン/サブエージェント/metaのファイルをまとめる。
	// dirty セッションの作り直しには、変わっていないファイルも含めた全体が要る。
	mainsOf := map[string][]scannedFile{}
	subsOf := map[string][]scannedFile{}
	metasOf := map[string][]scannedFile{}
	for _, f := range files {
		sid := sessionIDOfFile(f)
		if sid == "" {
			continue
		}
		switch f.Kind {
		case kindMain:
			mainsOf[sid] = append(mainsOf[sid], f)
		case kindSubAgent:
			subsOf[sid] = append(subsOf[sid], f)
		case kindMeta:
			metasOf[sid] = append(metasOf[sid], f)
		}
	}

	// 消えたファイルを先に落とす
	if len(removedPaths) != 0 {
		if err := c.removeFiles(removedPaths); err != nil {
			return err
		}
	}

	// dirty セッションをバッチで作り直す。1バッチ=1トランザクションなので、
	// 途中で失敗しても先行してコミット済みのバッチの行は残る。
	sessionList := make([]string, 0, len(dirtySessions))
	for sid := range dirtySessions {
		sessionList = append(sessionList, sid)
	}
	slices.Sort(sessionList) // 進捗表示とバッチ境界を安定させる
	c.setMeta("build_total_sessions", strconv.Itoa(len(sessionList)))
	c.setMeta("build_done_sessions", "0")
	if len(sessionList) != 0 {
		c.setMeta("build_state", "ingesting")
	}

	batchSize := buildBatchSessions
	if batchSize <= 0 {
		batchSize = 1
	}
	done := 0
	for start := 0; start < len(sessionList); start += batchSize {
		end := min(start+batchSize, len(sessionList))
		if err := c.ingestSessions(sessionList[start:end], mainsOf, subsOf, metasOf); err != nil {
			return err
		}
		done += end - start
		c.setMeta("build_done_sessions", strconv.Itoa(done))
	}

	// どのセッションにも属さない変化ファイル(kind=other など)の file_cache を進める。
	// 次の走査で「変わっていない」と判定させ、中身を読み直さないようにするため。
	if err := c.trackOrphanFiles(changedFiles, sessionIDOfFile); err != nil {
		return err
	}

	c.setMeta("build_state", "idle")
	c.setMeta("last_scan_unix", strconv.FormatInt(time.Now().Unix(), 10))
	return scanErr
}

// ingestSessions は指定セッションのメッセージを1トランザクションで作り直す。
func (c *pluginCache) ingestSessions(sids []string, mainsOf, subsOf, metasOf map[string][]scannedFile) error {
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

	for _, sid := range sids {
		if ingestSessionHook != nil {
			if err := ingestSessionHook(sid); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`DELETE FROM message_cache WHERE session_id = ?`, sid); err != nil {
			return fmt.Errorf("error at delete messages of session %s: %w", sid, err)
		}
		agentsByID, agentsByToolUseID := loadSubAgents(subsOf[sid], metasOf[sid])
		for _, mainFile := range mainsOf[sid] {
			records, rerr := readRecords(mainFile.Path)
			if rerr != nil && len(records) == 0 {
				continue
			}
			for _, t := range buildMessages(records, agentsByToolUseID, agentsByID) {
				if err := insertMessage(tx, mainFile.Path, t); err != nil {
					return err
				}
			}
		}
		// このセッションのファイルの file_cache を進める(セッションIDは sid で確定)
		for _, f := range sessionFiles(sid, mainsOf, subsOf, metasOf) {
			if err := upsertFileCache(tx, f, sid); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit ingest tx: %w", err)
	}
	committed = true
	return nil
}

// sessionFiles はセッションに属する全ファイル(メイン/サブ/meta)を集める。
func sessionFiles(sid string, mainsOf, subsOf, metasOf map[string][]scannedFile) []scannedFile {
	files := make([]scannedFile, 0, len(mainsOf[sid])+len(subsOf[sid])+len(metasOf[sid]))
	files = append(files, mainsOf[sid]...)
	files = append(files, subsOf[sid]...)
	files = append(files, metasOf[sid]...)
	return files
}

// trackOrphanFiles はどのセッションにも属さない変化ファイルの file_cache を進める。
func (c *pluginCache) trackOrphanFiles(changedFiles []scannedFile, sessionIDOfFile func(scannedFile) string) error {
	var orphans []scannedFile
	for _, f := range changedFiles {
		if sessionIDOfFile(f) == "" {
			orphans = append(orphans, f)
		}
	}
	if len(orphans) == 0 {
		return nil
	}

	db := c.conn()
	batchSize := buildBatchSessions
	if batchSize <= 0 {
		batchSize = 1
	}
	for start := 0; start < len(orphans); start += batchSize {
		end := min(start+batchSize, len(orphans))
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("error at begin orphan tx: %w", err)
		}
		for _, f := range orphans[start:end] {
			if err := upsertFileCache(tx, f, ""); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error at commit orphan tx: %w", err)
		}
	}
	return nil
}

// removeFiles は消えたファイルの発言と file_cache を落とす。
func (c *pluginCache) removeFiles(paths []string) error {
	db := c.conn()
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin remove tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, path := range paths {
		if _, err := tx.Exec(`DELETE FROM message_cache WHERE source_path = ?`, path); err != nil {
			return fmt.Errorf("error at delete messages of %s: %w", path, err)
		}
		if _, err := tx.Exec(`DELETE FROM file_cache WHERE path = ?`, path); err != nil {
			return fmt.Errorf("error at delete file cache %s: %w", path, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit remove tx: %w", err)
	}
	committed = true
	return nil
}

// loadSubAgents はセッションのサブエージェントを読み込み、
// エージェントID引きと tool_use.id 引きの2つのマップを返す。
func loadSubAgents(subs, metas []scannedFile) (byID, byToolUseID map[string]*subAgent) {
	byID = map[string]*subAgent{}
	byToolUseID = map[string]*subAgent{}

	metaByAgentID := map[string]agentMeta{}
	for _, m := range metas {
		agentID, meta, err := readAgentMeta(m.Path)
		if err != nil {
			continue
		}
		metaByAgentID[agentID] = meta
	}

	for _, s := range subs {
		agentID := agentIDFromFileName(s.Path)
		records, err := readRecords(s.Path)
		if err != nil && len(records) == 0 {
			continue
		}
		meta := metaByAgentID[agentID]
		sa := buildSubAgent(agentID, meta, records)
		byID[agentID] = sa
		if meta.ToolUseID != "" {
			byToolUseID[meta.ToolUseID] = sa
		}
	}
	return byID, byToolUseID
}

// insertMessage は1発言をキャッシュに書き込む。
func insertMessage(tx *sql.Tx, sourcePath string, t message) error {
	body, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("error at marshal message %s: %w", t.ID, err)
	}
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO message_cache(
			message_id, role, source_path, session_id, session_title, project, branch,
			message_text, search_text, body_json, related_time_unix, update_time_unix)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Role, sourcePath, t.SessionID, t.SessionTitle, t.Project, t.Branch,
		t.Text, searchTextOf(t), string(body), t.RelatedTime.Unix(), t.UpdateTime.Unix())
	if err != nil {
		return fmt.Errorf("error at insert message %s: %w", t.ID, err)
	}
	return nil
}

// upsertFileCache は走査結果の1ファイルを file_cache に書き込む。
func upsertFileCache(tx *sql.Tx, f scannedFile, sessionID string) error {
	if _, err := tx.Exec(
		`INSERT OR REPLACE INTO file_cache(path, mtime_unix, size, kind, session_id) VALUES(?,?,?,?,?)`,
		f.Path, f.MtimeUnix, f.Size, f.Kind, sessionID); err != nil {
		return fmt.Errorf("error at upsert file cache %s: %w", f.Path, err)
	}
	return nil
}

// loadFileCache は前回のスキャン結果を読み込む。
func (c *pluginCache) loadFileCache() (map[string]scannedFile, error) {
	db := c.conn()
	if db == nil {
		return nil, fmt.Errorf("cache db is not opened")
	}
	rows, err := db.Query(`SELECT path, mtime_unix, size, kind, session_id FROM file_cache`)
	if err != nil {
		return nil, fmt.Errorf("error at query file cache: %w", err)
	}
	defer func() { _ = rows.Close() }()

	known := map[string]scannedFile{}
	for rows.Next() {
		var f scannedFile
		if err := rows.Scan(&f.Path, &f.MtimeUnix, &f.Size, &f.Kind, &f.SessionID); err != nil {
			continue
		}
		known[f.Path] = f
	}
	return known, rows.Err()
}

// atoiOrZero は数値でない/空の meta 値を0にする。
func atoiOrZero(s string) int {
	value, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return value
}
