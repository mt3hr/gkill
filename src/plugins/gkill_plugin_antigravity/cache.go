package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	_ "modernc.org/sqlite"
)

// cacheSchemaVersion はDDLとパーサの世代。上げると次回の起動で作り直す。
//
// 差分更新は変わっていないファイルを二度と読まないので、パーサを直したときも上げること。
const cacheSchemaVersion = "1"

// buildBatchFiles は1トランザクションで取り込むファイル数。
// 途中で殺されても、このぶんだけやり直せばよい。
const buildBatchFiles = 16

// cache はキャッシュDBを持つ。プロセス内に1つ。
//
// ロックを3つに分けているのが要点。
//   - mu は db の遅延初期化だけを守る
//   - buildMu は構築どうしだけを直列化する
//   - 読み取り(query.go)はどちらも取らない
//
// 兼用すると初回構築のあいだ find_kyous が全部詰まり、
// gkill のデッドライン(IsAlive 5秒 / 呼び出し30秒)でプロセスが殺され続ける。
// SQLite は WAL で開いてあり *sql.DB 自体も並行安全なので、
// 構築中の読み取りは「そこまで取り込めたぶん」を返せばよい。
type cache struct {
	mu      sync.Mutex
	buildMu sync.Mutex
	db      *sql.DB
}

var globalCache = &cache{}

// openDB はキャッシュDBを開く。何度呼んでも1回しか開かない。
func (c *cache) openDB(pluginDir string) error {
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
func (c *cache) conn() *sql.DB {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.db
}

func initSchema(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS cache_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("error at create cache_meta: %w", err)
	}

	version := ""
	_ = db.QueryRow(`SELECT value FROM cache_meta WHERE key = 'schema_version'`).Scan(&version)
	if version != cacheSchemaVersion {
		for _, statement := range []string{
			`DROP TABLE IF EXISTS kyou_cache`,
			`DROP TABLE IF EXISTS conversation_meta`,
			`DROP TABLE IF EXISTS file_cache`,
			`DELETE FROM cache_meta`,
		} {
			if _, err := db.Exec(statement); err != nil {
				return fmt.Errorf("error at reset cache: %w", err)
			}
		}
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS file_cache (
  path            TEXT PRIMARY KEY,
  mtime_unix      INTEGER NOT NULL,
  size            INTEGER NOT NULL,
  kind            TEXT    NOT NULL,
  conversation_id TEXT    NOT NULL,
  step_count      INTEGER NOT NULL,
  kyou_count      INTEGER NOT NULL,
  dropped_lines   INTEGER NOT NULL,
  ingested_unix   INTEGER NOT NULL
)`,
		`CREATE INDEX IF NOT EXISTS idx_file_conversation ON file_cache(conversation_id)`,

		`CREATE TABLE IF NOT EXISTS conversation_meta (
  conversation_id TEXT PRIMARY KEY,
  title           TEXT NOT NULL,
  project         TEXT NOT NULL
) WITHOUT ROWID`,

		`CREATE TABLE IF NOT EXISTS kyou_cache (
  kyou_id         TEXT PRIMARY KEY,
  conversation_id TEXT    NOT NULL,
  role            TEXT    NOT NULL,
  step_index      INTEGER NOT NULL,
  title           TEXT    NOT NULL,
  project         TEXT    NOT NULL,
  model           TEXT    NOT NULL,
  search_text     TEXT    NOT NULL,
  body_json       TEXT    NOT NULL,
  related_unix    INTEGER NOT NULL,
  update_unix     INTEGER NOT NULL
)`,
		`CREATE INDEX IF NOT EXISTS idx_kyou_related      ON kyou_cache(related_unix DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_kyou_conversation ON kyou_cache(conversation_id)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("error at create cache table: %w", err)
		}
	}

	if _, err := db.Exec(
		`INSERT OR REPLACE INTO cache_meta (key, value) VALUES ('schema_version', ?)`,
		cacheSchemaVersion,
	); err != nil {
		return fmt.Errorf("error at store schema_version: %w", err)
	}
	return nil
}

func (c *cache) setMeta(key, value string) {
	db := c.conn()
	if db == nil {
		return
	}
	_, _ = db.Exec(`INSERT OR REPLACE INTO cache_meta (key, value) VALUES (?, ?)`, key, value)
}

func (c *cache) getMeta(key string) string {
	db := c.conn()
	if db == nil {
		return ""
	}
	value := ""
	_ = db.QueryRow(`SELECT value FROM cache_meta WHERE key = ?`, key).Scan(&value)
	return value
}

// loadFileCache は前回の走査結果を返す。
func (c *cache) loadFileCache() (map[string]scannedFile, error) {
	db := c.conn()
	if db == nil {
		return nil, fmt.Errorf("cache db is not opened")
	}
	rows, err := db.Query(`SELECT path, mtime_unix, size, kind, conversation_id FROM file_cache`)
	if err != nil {
		return nil, fmt.Errorf("error at select file_cache: %w", err)
	}
	defer func() { _ = rows.Close() }()

	known := map[string]scannedFile{}
	for rows.Next() {
		var file scannedFile
		if err := rows.Scan(&file.Path, &file.MtimeUnix, &file.Size, &file.Kind, &file.ConversationID); err != nil {
			return nil, fmt.Errorf("error at scan file_cache: %w", err)
		}
		known[file.Path] = file
	}
	return known, rows.Err()
}

// build はキャッシュを最新にする。ビルダ以外から呼ばないこと。
func (c *cache) build(pluginDir string, config pluginConfig) error {
	// 構築どうしだけを直列化する。読み取りは待たせない
	c.buildMu.Lock()
	defer c.buildMu.Unlock()

	if err := c.openDB(pluginDir); err != nil {
		return err
	}

	c.setMeta("build_state", "scanning")
	c.setMeta("build_error", "")

	source := expandPatterns(config.Patterns)
	c.storeStrings("source_problems", source.Missing)

	files, problems, scanErr := scanSources(source)
	c.storeStrings("missing_id_files", problems.MissingID)
	c.setMeta("shadowed_count", strconv.Itoa(problems.Shadowed))

	known, err := c.loadFileCache()
	if err != nil {
		return err
	}

	transcripts := make([]scannedFile, 0, len(files))
	summaries := make([]scannedFile, 0, 2)
	current := map[string]struct{}{}
	for _, file := range files {
		current[file.Path] = struct{}{}
		switch file.Kind {
		case kindTranscript:
			transcripts = append(transcripts, file)
		case kindSummaries:
			summaries = append(summaries, file)
		}
	}
	c.setMeta("target_file_count", strconv.Itoa(len(transcripts)))

	changedTranscripts := changedOnly(transcripts, known)
	removed := removedPaths(known, current)

	c.setMeta("build_total_files", strconv.Itoa(len(changedTranscripts)))
	c.setMeta("build_done_files", "0")

	// 会話タイトルを先に反映する。transcript は1バイトも読み直さない
	// (タイトルは kyou_cache の別カラムなので、取り込み直しが要らない)。
	//
	// summaries は (mtime, size) で差分を見ない。SQLite はページ単位で書くので、
	// タイトルが変わってもファイルのサイズは動かず、秒精度の mtime も同じ秒なら動かない
	// —— 差分を見ると「タイトルが付いたのに一覧に出ない」が起きる。
	// 代わりに毎回読む(行数ぶんの小さなクエリ1本で、値が実際に変わった会話だけ UPDATE する)。
	if len(summaries) != 0 {
		if err := c.refreshConversationMeta(summaries); err != nil {
			return err
		}
	}

	if len(removed) != 0 {
		if err := c.removeFiles(removed); err != nil {
			return err
		}
	}

	if len(changedTranscripts) != 0 {
		c.setMeta("build_state", "ingesting")
		if err := c.ingest(changedTranscripts); err != nil {
			return err
		}
	}

	c.setMeta("build_state", "idle")
	c.setMeta("last_scan_unix", strconv.FormatInt(time.Now().Unix(), 10))
	return scanErr
}

// changedOnly は (mtime, size) が前回と違うファイルだけを返す。
func changedOnly(files []scannedFile, known map[string]scannedFile) []scannedFile {
	changed := make([]scannedFile, 0, len(files))
	for _, file := range files {
		previous, exist := known[file.Path]
		if exist && previous.MtimeUnix == file.MtimeUnix && previous.Size == file.Size {
			continue
		}
		changed = append(changed, file)
	}
	return changed
}

// removedPaths は前回あって今回無いパスを返す。
func removedPaths(known map[string]scannedFile, current map[string]struct{}) []string {
	var removed []string
	for path := range known {
		if _, exist := current[path]; !exist {
			removed = append(removed, path)
		}
	}
	return removed
}

// refreshConversationMeta は conversation_summaries.db を読み直して会話タイトルを更新する。
//
// 値が実際に変わった会話だけ kyou_cache を1文で更新する。
// タイトルとプロジェクト名は search_text に焼き込んでいないので、取り込み直しは要らない。
// 読めなくても取り込みは続ける(Codex の session_index.jsonl と同じ任意扱い)。
func (c *cache) refreshConversationMeta(summaries []scannedFile) error {
	db := c.conn()
	metas := map[string]conversationMeta{}
	for _, file := range summaries {
		read, err := readConversationSummaries(file.Path)
		if err != nil {
			sdk.LogWarn("%s: read conversation summaries %s: %v", appName, file.Path, err)
			continue
		}
		maps.Copy(metas, read)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin conversation meta tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for conversationID, meta := range metas {
		previousTitle, previousProject := "", ""
		_ = tx.QueryRow(
			`SELECT title, project FROM conversation_meta WHERE conversation_id = ?`, conversationID,
		).Scan(&previousTitle, &previousProject)
		if previousTitle == meta.Title && previousProject == meta.Project {
			continue
		}
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO conversation_meta (conversation_id, title, project) VALUES (?, ?, ?)`,
			conversationID, meta.Title, meta.Project,
		); err != nil {
			return fmt.Errorf("error at store conversation_meta: %w", err)
		}
		if _, err := tx.Exec(
			`UPDATE kyou_cache SET title = ?, project = ? WHERE conversation_id = ?`,
			meta.Title, meta.Project, conversationID,
		); err != nil {
			return fmt.Errorf("error at update kyou title: %w", err)
		}
	}
	for _, file := range summaries {
		if err := upsertFileCache(tx, file, 0, 0, 0); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit conversation meta tx: %w", err)
	}
	committed = true
	return nil
}

// removeFiles は消えたファイルをキャッシュから落とす。
//
// transcript が消えた会話の Kyou は、その会話を指す transcript が
// ほかに1つも無くなったときだけ消す(同じ会話が2か所にあって勝ち負けが入れ替わったときに、
// 入れ替えのあいだだけ記録が消えないようにするため)。
func (c *cache) removeFiles(paths []string) error {
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
		conversationID := ""
		kind := ""
		_ = tx.QueryRow(`SELECT conversation_id, kind FROM file_cache WHERE path = ?`, path).Scan(&conversationID, &kind)
		if _, err := tx.Exec(`DELETE FROM file_cache WHERE path = ?`, path); err != nil {
			return fmt.Errorf("error at delete file_cache: %w", err)
		}
		if kind != kindTranscript || conversationID == "" {
			continue
		}
		remaining := 0
		_ = tx.QueryRow(
			`SELECT COUNT(*) FROM file_cache WHERE kind = ? AND conversation_id = ?`,
			kindTranscript, conversationID,
		).Scan(&remaining)
		if remaining != 0 {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM kyou_cache WHERE conversation_id = ?`, conversationID); err != nil {
			return fmt.Errorf("error at delete kyou_cache: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit remove tx: %w", err)
	}
	committed = true
	return nil
}

// ingest は変わった transcript を読み直して、その会話の Kyou を差し替える。
//
// 1ファイル=1会話なので、会話ごとに「全部消して入れ直す」でよい。
// 解析は逐次でよい(1ファイルは数百KB級で、Codex のロールアウトのように数十MBにならない)。
func (c *cache) ingest(changed []scannedFile) error {
	db := c.conn()
	droppedTotal := 0
	done := 0

	for start := 0; start < len(changed); start += buildBatchFiles {
		end := min(start+buildBatchFiles, len(changed))
		batch := changed[start:end]

		parsed := make([]parsedTranscript, 0, len(batch))
		used := make([]scannedFile, 0, len(batch))
		for _, file := range batch {
			result, err := parseTranscript(file)
			if err != nil {
				sdk.LogWarn("%s: parse %s: %v", appName, file.Path, err)
				continue
			}
			parsed = append(parsed, result)
			used = append(used, file)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("error at begin ingest tx: %w", err)
		}
		for i, result := range parsed {
			title, project := "", ""
			_ = tx.QueryRow(
				`SELECT title, project FROM conversation_meta WHERE conversation_id = ?`, result.ConversationID,
			).Scan(&title, &project)

			if _, err := tx.Exec(`DELETE FROM kyou_cache WHERE conversation_id = ?`, result.ConversationID); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("error at delete kyou_cache: %w", err)
			}
			title = titleOrFallback(title, result.Messages)
			for _, built := range result.Messages {
				built.Title = title
				built.Project = project
				if err := insertKyou(tx, built); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
			if err := upsertFileCache(tx, used[i], int64(result.StepCount),
				int64(len(result.Messages)), int64(result.DroppedLines)); err != nil {
				_ = tx.Rollback()
				return err
			}
			droppedTotal += result.DroppedLines
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error at commit ingest tx: %w", err)
		}

		done += len(batch)
		c.setMeta("build_done_files", strconv.Itoa(done))
	}

	if droppedTotal != 0 {
		sdk.LogWarn("%s: dropped %d unreadable lines", appName, droppedTotal)
	}
	return nil
}

// titleOrFallback は会話タイトルを決める。
//
// conversation_summaries.db が無い・まだ WAL にしか載っていない会話では
// タイトルが空になるので、最初の人間の発言の冒頭で代用する
// (一覧で会話を見分けられなくなるのを避ける)。
func titleOrFallback(title string, all []message) string {
	if title != "" {
		return title
	}
	for _, candidate := range all {
		if candidate.Role == roleHuman && candidate.Text != "" {
			return truncateRunes(firstLineOf(candidate.Text), 60)
		}
	}
	return ""
}

// firstLineOf は最初の空でない行を返す。
func firstLineOf(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// insertKyou は Kyou 1件をキャッシュへ入れる。
func insertKyou(tx *sql.Tx, built message) error {
	body, err := json.Marshal(built)
	if err != nil {
		return fmt.Errorf("error at marshal body %s: %w", built.ID, err)
	}
	updateUnix := built.UpdateTime.Unix()
	if built.UpdateTime.IsZero() {
		updateUnix = 0
	}
	relatedUnix := built.RelatedTime.Unix()
	if built.RelatedTime.IsZero() {
		relatedUnix = 0
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO kyou_cache
  (kyou_id, conversation_id, role, step_index, title, project, model, search_text, body_json,
   related_unix, update_unix)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		built.ID, built.ConversationID, built.Role, built.StepIndex, built.Title, built.Project,
		built.Model, searchTextOf(built), string(body), relatedUnix, updateUnix); err != nil {
		return fmt.Errorf("error at insert kyou_cache: %w", err)
	}
	return nil
}

// upsertFileCache は走査結果を控える。次回の差分判定に使う。
func upsertFileCache(tx *sql.Tx, file scannedFile, stepCount, kyouCount, dropped int64) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO file_cache
  (path, mtime_unix, size, kind, conversation_id, step_count, kyou_count, dropped_lines, ingested_unix)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		file.Path, file.MtimeUnix, file.Size, file.Kind, file.ConversationID,
		stepCount, kyouCount, dropped, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("error at upsert file_cache: %w", err)
	}
	return nil
}

// storeStrings は設定画面に出す文字列の一覧を cache_meta へ控える。
func (c *cache) storeStrings(key string, values []string) {
	if len(values) == 0 {
		c.setMeta(key, "")
		return
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return
	}
	c.setMeta(key, string(encoded))
}
