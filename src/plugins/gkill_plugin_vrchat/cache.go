package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	_ "modernc.org/sqlite"
)

// cacheSchemaVersion はDDLとパーサの世代。上げると次回の起動で作り直す。
//
// 差分更新は変わっていないファイルを二度と読まないので、パーサや記録の作り方を直したときも上げること。
const cacheSchemaVersion = "2"

// cache はキャッシュDBを持つ。プロセス内に1つ。
//
// ロックの分け方は gkill_plugin_codex と同じ（ADR-0305）。
//   - mu は db の遅延初期化だけを守る
//   - buildMu は構築どうしだけを直列化する
//   - 読み取り(query.go)はどちらも取らない
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
			`DROP TABLE IF EXISTS record`,
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
  path          TEXT PRIMARY KEY,
  mtime_unix    INTEGER NOT NULL,
  size          INTEGER NOT NULL,
  session_key   TEXT    NOT NULL,
  dropped_lines INTEGER NOT NULL,
  ingested_unix INTEGER NOT NULL
)`,
		`CREATE INDEX IF NOT EXISTS idx_file_session ON file_cache(session_key)`,

		`CREATE TABLE IF NOT EXISTS record (
  kyou_id      TEXT PRIMARY KEY,
  session_key  TEXT    NOT NULL,
  kind         TEXT    NOT NULL,
  title        TEXT    NOT NULL,
  start_unix   INTEGER NOT NULL,
  end_unix     INTEGER NOT NULL,
  url          TEXT    NOT NULL,
  description  TEXT    NOT NULL,
  content      TEXT    NOT NULL,
  tags         TEXT    NOT NULL,
  search_text  TEXT    NOT NULL,
  related_unix INTEGER NOT NULL,
  update_unix  INTEGER NOT NULL
)`,
		`CREATE INDEX IF NOT EXISTS idx_record_related ON record(related_unix DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_record_session ON record(session_key)`,
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
	rows, err := db.Query(`SELECT path, mtime_unix, size, session_key FROM file_cache`)
	if err != nil {
		return nil, fmt.Errorf("error at select file_cache: %w", err)
	}
	defer func() { _ = rows.Close() }()

	known := map[string]scannedFile{}
	for rows.Next() {
		var file scannedFile
		if err := rows.Scan(&file.Path, &file.MtimeUnix, &file.Size, &file.SessionKey); err != nil {
			return nil, fmt.Errorf("error at scan file_cache: %w", err)
		}
		known[file.Path] = file
	}
	return known, rows.Err()
}

// buildBatchFiles は1トランザクションで取り込むファイル数。
// 途中で殺されても、このぶんだけやり直せばよい。
const buildBatchFiles = 16

// buildBatchSessions は1トランザクションで記録を作り直す起動の数。
const buildBatchSessions = 32

// build はキャッシュを最新にする。ビルダ以外から呼ばないこと。
//
// 流れ:
//  1. 走査して (mtime, size) が変わったファイルと消えたファイルを出す
//  2. 変わったファイルを読み、起動キー（最初の時刻付きの行）を file_cache に控える
//  3. 変わった・消えたファイルが属する起動ごとに、記録を作り直す
//
// 同じ起動のログが複数の場所にある（VRChat の書き出し先と同期で運んだコピー、
// プレイ中に運んだ短い版と後の完全な版）ので、記録は起動キーごとに
// **一番大きいファイル**から作る。追記しかされないログなので、大きいほうが新しい。
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
	current := map[string]struct{}{}
	for _, file := range files {
		current[file.Path] = struct{}{}
	}
	c.setMeta("target_file_count", strconv.Itoa(len(files)))

	changed := changedOnly(files, known)
	removed := removedPaths(known, current)

	c.setMeta("build_total_files", strconv.Itoa(len(changed)))
	c.setMeta("build_done_files", "0")

	affected := map[string]struct{}{}
	parsedByPath := map[string]parsedLog{}

	if len(removed) != 0 {
		if err := c.removeFiles(removed); err != nil {
			return err
		}
		for _, path := range removed {
			affected[known[path].SessionKey] = struct{}{}
		}
	}

	if len(changed) != 0 {
		c.setMeta("build_state", "ingesting")
		if err := c.ingest(changed, known, affected, parsedByPath); err != nil {
			return err
		}
	}

	c.setMeta("build_state", "folding")
	if err := c.rebuildSessions(affected, parsedByPath); err != nil {
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
	sort.Strings(removed)
	return removed
}

// removeFiles は消えたファイルを file_cache から落とす。記録は rebuildSessions が作り直す。
func (c *cache) removeFiles(paths []string) error {
	db := c.conn()
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin remove tx: %w", err)
	}
	for _, path := range paths {
		if _, err := tx.Exec(`DELETE FROM file_cache WHERE path = ?`, path); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("error at delete file_cache: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit remove tx: %w", err)
	}
	return nil
}

// ingest は変わったファイルを読み、起動キーを file_cache に控える。
// 読んだ結果は parsedByPath に残し、記録の作り直しで同じファイルを2回読まない。
func (c *cache) ingest(changed []scannedFile, known map[string]scannedFile, affected map[string]struct{}, parsedByPath map[string]parsedLog) error {
	db := c.conn()
	done := 0
	for start := 0; start < len(changed); start += buildBatchFiles {
		end := min(start+buildBatchFiles, len(changed))
		batch := changed[start:end]

		// ログは1ファイル数百 KB 程度なので、解析は1本で足りる。DB書き込みはまとめて1トランザクション
		parsedBatch := make([]parsedLog, len(batch))
		for i, file := range batch {
			parsed, err := parseLogFile(file.Path)
			if err != nil {
				sdk.LogWarn("%s: parse %s: %v", appName, file.Path, err)
			}
			parsedBatch[i] = parsed
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("error at begin ingest tx: %w", err)
		}
		for i, file := range batch {
			parsed := parsedBatch[i]
			if _, err := tx.Exec(`INSERT OR REPLACE INTO file_cache
  (path, mtime_unix, size, session_key, dropped_lines, ingested_unix) VALUES (?, ?, ?, ?, ?, ?)`,
				file.Path, file.MtimeUnix, file.Size, parsed.SessionKey, parsed.Dropped, time.Now().Unix()); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("error at upsert file_cache: %w", err)
			}
			parsedByPath[file.Path] = parsed
			affected[parsed.SessionKey] = struct{}{}
			// 中身が書き換わって起動キーが変わったときは、前の起動の記録も作り直す
			if previous, exist := known[file.Path]; exist {
				affected[previous.SessionKey] = struct{}{}
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error at commit ingest tx: %w", err)
		}

		done += len(batch)
		c.setMeta("build_done_files", strconv.Itoa(done))
	}
	return nil
}

// rebuildSessions は起動キーごとに記録を作り直す。
// その起動のファイルのうち一番大きいもの（同じ大きさならパスの大きいほう）から作る。
func (c *cache) rebuildSessions(affected map[string]struct{}, parsedByPath map[string]parsedLog) error {
	db := c.conn()
	keys := make([]string, 0, len(affected))
	for key := range affected {
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	c.setMeta("dirty_session_count", strconv.Itoa(len(keys)))

	for start := 0; start < len(keys); start += buildBatchSessions {
		end := min(start+buildBatchSessions, len(keys))
		batch := keys[start:end]

		// 解析はトランザクションの外で済ませる
		recordsByKey := make(map[string][]record, len(batch))
		for _, key := range batch {
			winner := ""
			if err := db.QueryRow(`SELECT path FROM file_cache WHERE session_key = ?
  ORDER BY size DESC, path DESC LIMIT 1`, key).Scan(&winner); err != nil {
				if err == sql.ErrNoRows {
					recordsByKey[key] = nil // その起動のファイルが全部消えた
					continue
				}
				return fmt.Errorf("error at select winner of %s: %w", key, err)
			}
			parsed, ok := parsedByPath[winner]
			if !ok {
				read, err := parseLogFile(winner)
				if err != nil {
					sdk.LogWarn("%s: parse %s: %v", appName, winner, err)
					continue // 読めないときは前回の記録を残す
				}
				parsed = read
			}
			if parsed.SessionKey != key {
				// 走査と読み直しの間に中身が変わった。次の構築に任せる
				continue
			}
			recordsByKey[key] = buildRecords(parsed)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("error at begin rebuild tx: %w", err)
		}
		for _, key := range batch {
			records, rebuild := recordsByKey[key]
			if !rebuild {
				continue
			}
			if _, err := tx.Exec(`DELETE FROM record WHERE session_key = ?`, key); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("error at delete record: %w", err)
			}
			for _, r := range records {
				if err := insertRecord(tx, r); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error at commit rebuild tx: %w", err)
		}
	}
	c.setMeta("dirty_session_count", "0")
	return nil
}

func insertRecord(tx *sql.Tx, r record) error {
	endUnix := int64(0)
	if !r.End.IsZero() {
		endUnix = r.End.Unix()
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO record
  (kyou_id, session_key, kind, title, start_unix, end_unix, url, description, content, tags,
   search_text, related_unix, update_unix)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.SessionKey, r.Kind, r.Title, r.Start.Unix(), endUnix, r.URL, r.Description, r.Content,
		strings.Join(r.Tags, "\n"), r.SearchText, r.Start.Unix(), r.updateTime().Unix()); err != nil {
		return fmt.Errorf("error at insert record: %w", err)
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
