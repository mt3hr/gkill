package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// cacheSchemaVersion はDDLの世代。上げると次回の起動で作り直す。
const cacheSchemaVersion = "1"

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

	dbPath := cacheDBPath(pluginDir)

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
			`DROP TABLE IF EXISTS thread_item`,
			`DROP TABLE IF EXISTS thread_title`,
			`DROP TABLE IF EXISTS dirty_thread`,
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
  path             TEXT PRIMARY KEY,
  mtime_unix       INTEGER NOT NULL,
  size             INTEGER NOT NULL,
  kind             TEXT    NOT NULL,
  thread_id        TEXT    NOT NULL,
  parent_thread_id TEXT    NOT NULL,
  is_subagent      INTEGER NOT NULL,
  agent_path       TEXT    NOT NULL,
  agent_nickname   TEXT    NOT NULL,
  cwd              TEXT    NOT NULL,
  branch           TEXT    NOT NULL,
  repo_url         TEXT    NOT NULL,
  originator       TEXT    NOT NULL,
  cli_version      TEXT    NOT NULL,
  item_count       INTEGER NOT NULL,
  user_count       INTEGER NOT NULL,
  first_user_unix  INTEGER NOT NULL,
  dropped_lines    INTEGER NOT NULL,
  ingested_unix    INTEGER NOT NULL
)`,
		`CREATE INDEX IF NOT EXISTS idx_file_thread ON file_cache(thread_id)`,
		`CREATE INDEX IF NOT EXISTS idx_file_parent ON file_cache(parent_thread_id)`,

		`CREATE TABLE IF NOT EXISTS thread_item (
  thread_id TEXT    NOT NULL,
  seq       INTEGER NOT NULL,
  ts_unix   INTEGER NOT NULL,
  kind      TEXT    NOT NULL,
  name      TEXT    NOT NULL,
  text      TEXT    NOT NULL,
  ref_id    TEXT    NOT NULL,
  extra     TEXT    NOT NULL,
  PRIMARY KEY (thread_id, seq)
) WITHOUT ROWID`,

		`CREATE TABLE IF NOT EXISTS dirty_thread (
  thread_id TEXT PRIMARY KEY
) WITHOUT ROWID`,

		`CREATE TABLE IF NOT EXISTS thread_title (
  thread_id TEXT PRIMARY KEY,
  title     TEXT NOT NULL
) WITHOUT ROWID`,

		`CREATE TABLE IF NOT EXISTS kyou_cache (
  kyou_id        TEXT PRIMARY KEY,
  root_thread_id TEXT    NOT NULL,
  thread_id      TEXT    NOT NULL,
  role           TEXT    NOT NULL,
  ordinal        INTEGER NOT NULL,
  title          TEXT    NOT NULL,
  project        TEXT    NOT NULL,
  branch         TEXT    NOT NULL,
  model          TEXT    NOT NULL,
  originator     TEXT    NOT NULL,
  message_text   TEXT    NOT NULL,
  search_text    TEXT    NOT NULL,
  body_json      TEXT    NOT NULL,
  related_unix   INTEGER NOT NULL,
  update_unix    INTEGER NOT NULL
)`,
		`CREATE INDEX IF NOT EXISTS idx_kyou_related ON kyou_cache(related_unix DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_kyou_root    ON kyou_cache(root_thread_id)`,
		`CREATE INDEX IF NOT EXISTS idx_kyou_thread  ON kyou_cache(thread_id)`,
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
	rows, err := db.Query(`SELECT path, mtime_unix, size, kind, thread_id, parent_thread_id,
  is_subagent, agent_path, agent_nickname, cwd, branch, repo_url, originator, cli_version,
  user_count, first_user_unix
FROM file_cache`)
	if err != nil {
		return nil, fmt.Errorf("error at select file_cache: %w", err)
	}
	defer func() { _ = rows.Close() }()

	known := map[string]scannedFile{}
	for rows.Next() {
		var file scannedFile
		var isSubAgent int
		var userCount, firstUserUnix int64
		if err := rows.Scan(&file.Path, &file.MtimeUnix, &file.Size, &file.Kind,
			&file.Meta.ThreadID, &file.Meta.ParentThreadID, &isSubAgent,
			&file.Meta.AgentPath, &file.Meta.AgentNickname, &file.Meta.Cwd,
			&file.Meta.Branch, &file.Meta.RepoURL, &file.Meta.Originator, &file.Meta.CLIVersion,
			&userCount, &firstUserUnix); err != nil {
			return nil, fmt.Errorf("error at scan file_cache: %w", err)
		}
		file.Meta.IsSubAgent = isSubAgent != 0
		file.userCount = userCount
		file.firstUserUnix = firstUserUnix
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
	c.storeSourceProblems(source.Missing)

	files, scanErr := scanSources(source)
	known, err := c.loadFileCache()
	if err != nil {
		return err
	}

	rollouts := make([]scannedFile, 0, len(files))
	indexes := make([]scannedFile, 0, 2)
	current := map[string]struct{}{}
	for _, file := range files {
		current[file.Path] = struct{}{}
		switch file.Kind {
		case kindRollout:
			rollouts = append(rollouts, file)
		case kindIndex:
			indexes = append(indexes, file)
		}
	}
	c.setMeta("target_file_count", strconv.Itoa(len(rollouts)))

	changedIndexes := changedOnly(indexes, known)
	changed := changedOnly(rollouts, known)
	removed := removedPaths(known, current)

	c.setMeta("build_total_files", strconv.Itoa(len(changed)))
	c.setMeta("build_done_files", "0")

	// スレッド名を先に反映する。ロールアウトは1バイトも読み直さない
	if len(changedIndexes) != 0 {
		if err := c.refreshTitles(changedIndexes); err != nil {
			return err
		}
	}

	if len(removed) != 0 {
		if err := c.removeFiles(removed, known); err != nil {
			return err
		}
	}

	if len(changed) != 0 {
		c.setMeta("build_state", "ingesting")
		if err := c.ingest(changed, known, config); err != nil {
			return err
		}
	}

	c.setMeta("build_state", "folding")
	if err := c.foldDirty(config.SubagentMode); err != nil {
		return err
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

func removedPaths(known map[string]scannedFile, current map[string]struct{}) []string {
	var removed []string
	for path := range known {
		if _, exist := current[path]; !exist {
			removed = append(removed, path)
		}
	}
	return removed
}

// refreshTitles は session_index.jsonl を読み直してスレッド名を更新する。
//
// 値が実際に変わったスレッドだけ kyou_cache を1文で更新する。
// title を search_text に焼き込んでいないので、畳み直しは要らない。
func (c *cache) refreshTitles(indexes []scannedFile) error {
	db := c.conn()
	titles := map[string]string{}
	for _, file := range indexes {
		read, err := readSessionIndex(file.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: read session index %s: %v\n", appName, file.Path, err)
			continue
		}
		for threadID, title := range read {
			titles[threadID] = title
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin title tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for threadID, title := range titles {
		previous := ""
		_ = tx.QueryRow(`SELECT title FROM thread_title WHERE thread_id = ?`, threadID).Scan(&previous)
		if previous == title {
			continue
		}
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO thread_title (thread_id, title) VALUES (?, ?)`,
			threadID, title,
		); err != nil {
			return fmt.Errorf("error at store thread_title: %w", err)
		}
		if _, err := tx.Exec(
			`UPDATE kyou_cache SET title = ? WHERE thread_id = ?`, title, threadID,
		); err != nil {
			return fmt.Errorf("error at update kyou title: %w", err)
		}
	}
	for _, file := range indexes {
		if err := upsertFileCache(tx, file, 0, 0, 0, 0); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit title tx: %w", err)
	}
	committed = true
	return nil
}

// removeFiles は消えたファイルをキャッシュから落とす。
func (c *cache) removeFiles(paths []string, known map[string]scannedFile) error {
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
		previous := known[path]
		if previous.Meta.ThreadID != "" {
			if _, err := tx.Exec(`DELETE FROM thread_item WHERE thread_id = ?`, previous.Meta.ThreadID); err != nil {
				return fmt.Errorf("error at delete thread_item: %w", err)
			}
			// 自分と、居なくなったことで畳み直しが要る親の両方を積む
			if err := markDirty(tx, previous.Meta.ThreadID, previous.Meta.ParentThreadID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`DELETE FROM file_cache WHERE path = ?`, path); err != nil {
			return fmt.Errorf("error at delete file_cache: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit remove tx: %w", err)
	}
	committed = true
	return nil
}

// buildBatchFiles は1トランザクションで取り込むファイル数。
// 途中で殺されても、このぶんだけやり直せばよい。
const buildBatchFiles = 16

// foldBatchRoots は1トランザクションで畳み直すスレッド木の数。
const foldBatchRoots = 32

// ingest は変わったロールアウトを読み直して thread_item を差し替える。
func (c *cache) ingest(changed []scannedFile, known map[string]scannedFile, config pluginConfig) error {
	db := c.conn()

	workers := config.ScanWorkers
	if workers <= 0 {
		workers = min(max(runtime.NumCPU()/2, 1), 4)
	}

	droppedTotal := 0
	unknownKinds := map[string]struct{}{}
	rewrites := []string{}
	done := 0

	for start := 0; start < len(changed); start += buildBatchFiles {
		end := min(start+buildBatchFiles, len(changed))
		batch := changed[start:end]

		// 解析は並列、DB書き込みは1本にまとめる
		results := make([]parsedRollout, len(batch))
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, workers)
		for i, file := range batch {
			wg.Add(1)
			go func(i int, file scannedFile) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				parsed, err := parseRollout(file.Path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s: parse %s: %v\n", appName, file.Path, err)
					return
				}
				results[i] = parsed
			}(i, file)
		}
		wg.Wait()

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("error at begin ingest tx: %w", err)
		}
		committed := false
		for i, file := range batch {
			parsed := results[i]
			// ファイル名のuuidを正とする。session_meta.id は突き合わせにだけ使う
			threadID := threadIDFromFileName(file.Path)
			if threadID == "" {
				threadID = strings.ToLower(parsed.Meta.ThreadID)
			}
			if threadID == "" || len(parsed.Items) == 0 && parsed.Meta.ThreadID == "" {
				// session_meta が無い＝ロールアウトではない。記録だけして中身は持たない
				continue
			}
			parsed.Meta.ThreadID = threadID
			parsed.Meta.ParentThreadID = strings.ToLower(parsed.Meta.ParentThreadID)
			file.Meta = parsed.Meta

			userCount, firstUserUnix := userStatsOf(parsed.Items)
			if previous, exist := known[file.Path]; exist {
				if reason := rewriteReason(previous, file.Size, userCount, firstUserUnix); reason != "" {
					rewrites = append(rewrites, threadID+": "+reason)
				}
			}

			if _, err := tx.Exec(`DELETE FROM thread_item WHERE thread_id = ?`, threadID); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("error at delete thread_item: %w", err)
			}
			if err := insertItems(tx, threadID, parsed.Items); err != nil {
				_ = tx.Rollback()
				return err
			}
			if err := upsertFileCache(tx, file, int64(len(parsed.Items)), userCount, firstUserUnix, int64(parsed.Dropped)); err != nil {
				_ = tx.Rollback()
				return err
			}
			// 新しい親と、前回の親(親が付け替わったとき)の両方を畳み直し対象にする
			previousParent := ""
			if previous, exist := known[file.Path]; exist {
				previousParent = previous.Meta.ParentThreadID
			}
			if err := markDirty(tx, threadID, parsed.Meta.ParentThreadID, previousParent); err != nil {
				_ = tx.Rollback()
				return err
			}

			droppedTotal += parsed.Dropped
			for _, kind := range parsed.UnknownKinds {
				unknownKinds[kind] = struct{}{}
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error at commit ingest tx: %w", err)
		}
		committed = true
		_ = committed

		done += len(batch)
		c.setMeta("build_done_files", strconv.Itoa(done))
	}

	if droppedTotal > 0 {
		c.setMeta("dropped_lines_total", strconv.Itoa(droppedTotal))
	}
	if len(unknownKinds) != 0 {
		kinds := make([]string, 0, len(unknownKinds))
		for kind := range unknownKinds {
			kinds = append(kinds, kind)
		}
		c.setMeta("unknown_kinds", strings.Join(dedupeStrings(kinds), ", "))
	}
	if len(rewrites) != 0 {
		c.setMeta("rewrite_warning", strings.Join(rewrites, " / "))
	}
	return nil
}

// userStatsOf は人間の発言の数と、最初の発言の時刻を返す。書き換え検出に使う。
func userStatsOf(items []threadItem) (int64, int64) {
	count := int64(0)
	first := int64(0)
	for _, item := range items {
		if item.Kind != itemUser {
			continue
		}
		count++
		if first == 0 && !item.TS.IsZero() {
			first = item.TS.Unix()
		}
	}
	return count, first
}

// rewriteReason は「追記のみ」の前提が破れた疑いを説明する。空なら疑い無し。
//
// 破れると ordinal がずれて KyouID が変わり、ユーザが付けたタグやテキストが
// 迷子になる。自動修復はせず、設定画面に出して気づけるようにするだけ。
func rewriteReason(previous scannedFile, size, userCount, firstUserUnix int64) string {
	switch {
	case size < previous.Size:
		return "ファイルが小さくなりました"
	case previous.userCount > 0 && userCount < previous.userCount:
		return "発言の数が減りました"
	case previous.firstUserUnix != 0 && firstUserUnix != 0 && previous.firstUserUnix != firstUserUnix:
		return "最初の発言の時刻が変わりました"
	}
	return ""
}

func insertItems(tx *sql.Tx, threadID string, items []threadItem) error {
	if len(items) == 0 {
		return nil
	}
	statement, err := tx.Prepare(`INSERT OR REPLACE INTO thread_item
  (thread_id, seq, ts_unix, kind, name, text, ref_id, extra) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("error at prepare thread_item: %w", err)
	}
	defer func() { _ = statement.Close() }()

	for _, item := range items {
		ts := int64(0)
		if !item.TS.IsZero() {
			ts = item.TS.Unix()
		}
		if _, err := statement.Exec(threadID, item.Seq, ts,
			item.Kind, item.Name, item.Text, item.RefID, item.Extra); err != nil {
			return fmt.Errorf("error at insert thread_item: %w", err)
		}
	}
	return nil
}

func upsertFileCache(tx *sql.Tx, file scannedFile, itemCount, userCount, firstUserUnix, dropped int64) error {
	isSubAgent := 0
	if file.Meta.IsSubAgent {
		isSubAgent = 1
	}
	_, err := tx.Exec(`INSERT OR REPLACE INTO file_cache
  (path, mtime_unix, size, kind, thread_id, parent_thread_id, is_subagent,
   agent_path, agent_nickname, cwd, branch, repo_url, originator, cli_version,
   item_count, user_count, first_user_unix, dropped_lines, ingested_unix)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		file.Path, file.MtimeUnix, file.Size, file.Kind,
		file.Meta.ThreadID, file.Meta.ParentThreadID, isSubAgent,
		file.Meta.AgentPath, file.Meta.AgentNickname, file.Meta.Cwd,
		file.Meta.Branch, file.Meta.RepoURL, file.Meta.Originator, file.Meta.CLIVersion,
		itemCount, userCount, firstUserUnix, dropped, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("error at upsert file_cache: %w", err)
	}
	return nil
}

// markDirty は畳み直しが要るスレッドを積む。
// ルートの解決は畳み直しのときに行う(取り込み中はまだ親が揃っていないことがある)。
func markDirty(tx *sql.Tx, threadIDs ...string) error {
	for _, threadID := range threadIDs {
		if threadID == "" {
			continue
		}
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO dirty_thread (thread_id) VALUES (?)`, threadID,
		); err != nil {
			return fmt.Errorf("error at mark dirty: %w", err)
		}
	}
	return nil
}

// foldDirty は積まれたスレッドが属する木を畳み直す。
func (c *cache) foldDirty(mode string) error {
	db := c.conn()

	dirty, err := c.loadDirtyThreads()
	if err != nil {
		return err
	}
	if len(dirty) == 0 {
		c.setMeta("dirty_thread_count", "0")
		return nil
	}

	files, err := c.loadFileCache()
	if err != nil {
		return err
	}

	byThread := map[string]scannedFile{}
	parents := map[string]string{}
	known := map[string]struct{}{}
	for _, file := range files {
		if file.Kind != kindRollout || file.Meta.ThreadID == "" {
			continue
		}
		byThread[file.Meta.ThreadID] = file
		parents[file.Meta.ThreadID] = file.Meta.ParentThreadID
		known[file.Meta.ThreadID] = struct{}{}
	}

	childrenOf := map[string][]string{}
	for threadID := range known {
		root := rootOf(threadID, parents, known)
		if root != threadID {
			childrenOf[root] = append(childrenOf[root], threadID)
		}
	}

	roots := map[string]struct{}{}
	for _, threadID := range dirty {
		if _, exist := known[threadID]; exist {
			roots[rootOf(threadID, parents, known)] = struct{}{}
			continue
		}
		// 消えたスレッド。自分がルートだった場合に備えて掃除対象にする
		roots[threadID] = struct{}{}
	}

	titles, err := c.loadTitles()
	if err != nil {
		return err
	}

	rootList := make([]string, 0, len(roots))
	for root := range roots {
		rootList = append(rootList, root)
	}
	c.setMeta("dirty_thread_count", strconv.Itoa(len(rootList)))

	for start := 0; start < len(rootList); start += foldBatchRoots {
		end := min(start+foldBatchRoots, len(rootList))
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("error at begin fold tx: %w", err)
		}
		committed := false
		for _, root := range rootList[start:end] {
			if _, err := tx.Exec(`DELETE FROM kyou_cache WHERE root_thread_id = ?`, root); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("error at delete kyou_cache: %w", err)
			}
			if _, exist := byThread[root]; !exist {
				continue
			}
			group, err := c.buildGroup(tx, root, byThread, childrenOf[root], titles)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
			for _, built := range foldGroup(group, mode) {
				if err := insertKyou(tx, built); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
		if _, err := tx.Exec(`DELETE FROM dirty_thread`); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("error at clear dirty_thread: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error at commit fold tx: %w", err)
		}
		committed = true
		_ = committed
	}

	c.setMeta("dirty_thread_count", "0")
	return nil
}

func (c *cache) loadDirtyThreads() ([]string, error) {
	db := c.conn()
	rows, err := db.Query(`SELECT thread_id FROM dirty_thread`)
	if err != nil {
		return nil, fmt.Errorf("error at select dirty_thread: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var dirty []string
	for rows.Next() {
		threadID := ""
		if err := rows.Scan(&threadID); err != nil {
			return nil, fmt.Errorf("error at scan dirty_thread: %w", err)
		}
		dirty = append(dirty, threadID)
	}
	return dirty, rows.Err()
}

func (c *cache) loadTitles() (map[string]string, error) {
	db := c.conn()
	rows, err := db.Query(`SELECT thread_id, title FROM thread_title`)
	if err != nil {
		return nil, fmt.Errorf("error at select thread_title: %w", err)
	}
	defer func() { _ = rows.Close() }()

	titles := map[string]string{}
	for rows.Next() {
		threadID, title := "", ""
		if err := rows.Scan(&threadID, &title); err != nil {
			return nil, fmt.Errorf("error at scan thread_title: %w", err)
		}
		titles[threadID] = title
	}
	return titles, rows.Err()
}

// buildGroup は畳み直しに必要な要素をDBから集める。
func (c *cache) buildGroup(tx *sql.Tx, root string, byThread map[string]scannedFile, children []string, titles map[string]string) (threadGroup, error) {
	group := threadGroup{
		RootID:   root,
		Files:    map[string]scannedFile{},
		Items:    map[string][]threadItem{},
		Titles:   titles,
		Children: children,
	}
	for _, threadID := range append([]string{root}, children...) {
		file, exist := byThread[threadID]
		if !exist {
			continue
		}
		group.Files[threadID] = file
		items, err := loadItems(tx, threadID)
		if err != nil {
			return group, err
		}
		group.Items[threadID] = items
	}
	return group, nil
}

func loadItems(tx *sql.Tx, threadID string) ([]threadItem, error) {
	rows, err := tx.Query(
		`SELECT seq, ts_unix, kind, name, text, ref_id, extra FROM thread_item
		 WHERE thread_id = ? ORDER BY seq`, threadID)
	if err != nil {
		return nil, fmt.Errorf("error at select thread_item: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []threadItem
	for rows.Next() {
		var item threadItem
		var ts int64
		if err := rows.Scan(&item.Seq, &ts, &item.Kind, &item.Name, &item.Text, &item.RefID, &item.Extra); err != nil {
			return nil, fmt.Errorf("error at scan thread_item: %w", err)
		}
		if ts != 0 {
			item.TS = time.Unix(ts, 0).UTC()
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func insertKyou(tx *sql.Tx, built message) error {
	body, err := json.Marshal(built)
	if err != nil {
		return fmt.Errorf("error at marshal message: %w", err)
	}
	related := int64(0)
	if !built.RelatedTime.IsZero() {
		related = built.RelatedTime.Unix()
	}
	update := related
	if !built.UpdateTime.IsZero() {
		update = built.UpdateTime.Unix()
	}

	if _, err := tx.Exec(`INSERT OR REPLACE INTO kyou_cache
  (kyou_id, root_thread_id, thread_id, role, ordinal, title, project, branch, model,
   originator, message_text, search_text, body_json, related_unix, update_unix)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		built.ID, built.RootThreadID, built.ThreadID, built.Role, built.Ordinal,
		built.Title, built.Project, built.Branch, built.Model, built.Originator,
		built.Text, searchTextOf(built), string(body), related, update); err != nil {
		return fmt.Errorf("error at insert kyou_cache: %w", err)
	}
	return nil
}

func (c *cache) storeSourceProblems(missing []string) {
	if len(missing) == 0 {
		c.setMeta("source_problems", "")
		return
	}
	encoded, err := json.Marshal(missing)
	if err != nil {
		return
	}
	c.setMeta("source_problems", string(encoded))
}
