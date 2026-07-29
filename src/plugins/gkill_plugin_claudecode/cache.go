package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// pluginCache はトランスクリプトから組み立てたターンをSQLite3にキャッシュする。
// {pluginDir}/cache.db に保存し、ファイル単位のmtime/サイズで差分更新する。
// ソースは146MB規模になるため、毎回全部を読み直さないことが重要。
type pluginCache struct {
	mu sync.Mutex
	db *sql.DB
}

var globalCache = &pluginCache{}

// turnSummary はFindKyous用の軽量な行。body_jsonは読まない。
type turnSummary struct {
	TurnID          string
	SessionID       string
	SessionTitle    string
	Project         string
	Branch          string
	SearchText      string
	RelatedTimeUnix int64
	UpdateTimeUnix  int64
}

// cacheStats は設定画面に出す統計。
type cacheStats struct {
	FileCount     int
	TurnCount     int
	LastScanUnix  int64
	LastScanError string
}

// openDB はキャッシュDBを開く(初回は初期化する)。
func (c *pluginCache) openDB(pluginDir string) error {
	if c.db != nil {
		return nil
	}
	dbPath := filepath.Join(pluginDir, "cache.db")
	db, err := sqlite3impl.GetSQLiteDBConnection(context.Background(), dbPath)
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

// initSchema はテーブルを作成する。
func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS cache_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS file_cache (
  path       TEXT PRIMARY KEY,
  mtime_unix INTEGER NOT NULL,
  size       INTEGER NOT NULL,
  kind       TEXT NOT NULL,
  session_id TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS turn_cache (
  turn_id           TEXT PRIMARY KEY,
  source_path       TEXT NOT NULL,
  session_id        TEXT NOT NULL,
  session_title     TEXT NOT NULL,
  project           TEXT NOT NULL,
  branch            TEXT NOT NULL,
  prompt_text       TEXT NOT NULL,
  search_text       TEXT NOT NULL,
  body_json         TEXT NOT NULL,
  related_time_unix INTEGER NOT NULL,
  update_time_unix  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_turn_time    ON turn_cache(related_time_unix);
CREATE INDEX IF NOT EXISTS idx_turn_session ON turn_cache(session_id);
CREATE INDEX IF NOT EXISTS idx_turn_src     ON turn_cache(source_path);
`)
	if err != nil {
		return fmt.Errorf("error at init schema: %w", err)
	}
	return nil
}

// GetTurns はFindKyous用に全ターンの要約を返す。呼び出し前に差分更新する。
func (c *pluginCache) GetTurns(pluginDir string, src expandedSource) ([]turnSummary, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.openDB(pluginDir); err != nil {
		return nil, err
	}
	if err := c.refresh(src); err != nil {
		return nil, err
	}

	rows, err := c.db.Query(`
		SELECT turn_id, session_id, session_title, project, branch, search_text,
		       related_time_unix, update_time_unix
		FROM turn_cache ORDER BY related_time_unix DESC`)
	if err != nil {
		return nil, fmt.Errorf("error at query turns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var turns []turnSummary
	for rows.Next() {
		var t turnSummary
		if err := rows.Scan(&t.TurnID, &t.SessionID, &t.SessionTitle, &t.Project, &t.Branch,
			&t.SearchText, &t.RelatedTimeUnix, &t.UpdateTimeUnix); err != nil {
			continue
		}
		turns = append(turns, t)
	}
	return turns, nil
}

// GetTurn はGetContentHTML用に、ターン1件を本文込みで返す。
func (c *pluginCache) GetTurn(pluginDir string, src expandedSource, turnID string) (turn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var t turn
	if err := c.openDB(pluginDir); err != nil {
		return t, err
	}
	if err := c.refresh(src); err != nil {
		return t, err
	}

	var bodyJSON string
	row := c.db.QueryRow(`SELECT body_json FROM turn_cache WHERE turn_id = ?`, turnID)
	if err := row.Scan(&bodyJSON); err != nil {
		return t, fmt.Errorf("turn not found: %s", turnID)
	}
	if err := json.Unmarshal([]byte(bodyJSON), &t); err != nil {
		return t, fmt.Errorf("error at parse cached turn %s: %w", turnID, err)
	}
	return t, nil
}

// GetStats は設定画面用の統計を返す。
func (c *pluginCache) GetStats(pluginDir string, src expandedSource) cacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats := cacheStats{}
	if err := c.openDB(pluginDir); err != nil {
		stats.LastScanError = err.Error()
		return stats
	}
	if err := c.refresh(src); err != nil {
		stats.LastScanError = err.Error()
	}

	_ = c.db.QueryRow(`SELECT COUNT(*) FROM file_cache WHERE kind != ?`, kindOther).Scan(&stats.FileCount)
	_ = c.db.QueryRow(`SELECT COUNT(*) FROM turn_cache`).Scan(&stats.TurnCount)
	var lastScan string
	if err := c.db.QueryRow(`SELECT value FROM cache_meta WHERE key = 'last_scan_unix'`).Scan(&lastScan); err == nil {
		stats.LastScanUnix, _ = strconv.ParseInt(lastScan, 10, 64)
	}
	return stats
}

// refresh はソースを走査し、変化のあったセッションだけキャッシュを作り直す。
func (c *pluginCache) refresh(src expandedSource) error {
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
	for _, f := range files {
		current[f.Path] = f
		prev, ok := known[f.Path]
		if ok && prev.MtimeUnix == f.MtimeUnix && prev.Size == f.Size {
			continue
		}
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

	if len(dirtySessions) == 0 && len(removedPaths) == 0 {
		return scanErr
	}

	// セッションごとに、メイン/サブエージェント/metaのファイルをまとめる
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

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, path := range removedPaths {
		if _, err := tx.Exec(`DELETE FROM turn_cache WHERE source_path = ?`, path); err != nil {
			return fmt.Errorf("error at delete turns of %s: %w", path, err)
		}
		if _, err := tx.Exec(`DELETE FROM file_cache WHERE path = ?`, path); err != nil {
			return fmt.Errorf("error at delete file cache %s: %w", path, err)
		}
	}

	for sid := range dirtySessions {
		if _, err := tx.Exec(`DELETE FROM turn_cache WHERE session_id = ?`, sid); err != nil {
			return fmt.Errorf("error at delete turns of session %s: %w", sid, err)
		}
		agentsByID, agentsByToolUseID := loadSubAgents(subsOf[sid], metasOf[sid])
		for _, mainFile := range mainsOf[sid] {
			records, rerr := readRecords(mainFile.Path)
			if rerr != nil && len(records) == 0 {
				continue
			}
			for _, t := range buildTurns(records, agentsByToolUseID, agentsByID) {
				if err := insertTurn(tx, mainFile.Path, t); err != nil {
					return err
				}
			}
		}
	}

	for _, f := range files {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO file_cache(path, mtime_unix, size, kind, session_id) VALUES(?,?,?,?,?)`,
			f.Path, f.MtimeUnix, f.Size, f.Kind, sessionIDOfFile(f)); err != nil {
			return fmt.Errorf("error at upsert file cache %s: %w", f.Path, err)
		}
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO cache_meta(key,value) VALUES('last_scan_unix', ?)`,
		strconv.FormatInt(time.Now().Unix(), 10)); err != nil {
		return fmt.Errorf("error at update last scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit: %w", err)
	}
	committed = true
	return scanErr
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

// insertTurn は1ターンをキャッシュに書き込む。
func insertTurn(tx *sql.Tx, sourcePath string, t turn) error {
	body, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("error at marshal turn %s: %w", t.ID, err)
	}
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO turn_cache(
			turn_id, source_path, session_id, session_title, project, branch,
			prompt_text, search_text, body_json, related_time_unix, update_time_unix)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, sourcePath, t.SessionID, t.SessionTitle, t.Project, t.Branch,
		t.Prompt, searchTextOf(t), string(body), t.RelatedTime.Unix(), t.UpdateTime.Unix())
	if err != nil {
		return fmt.Errorf("error at insert turn %s: %w", t.ID, err)
	}
	return nil
}

// loadFileCache は前回のスキャン結果を読み込む。
func (c *pluginCache) loadFileCache() (map[string]scannedFile, error) {
	rows, err := c.db.Query(`SELECT path, mtime_unix, size, kind, session_id FROM file_cache`)
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
	return known, nil
}
