package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 読み取りは buildMu を取らない。構築中でも「そこまで取り込めたぶん」を返す。
// これが仕様どおりの挙動で、初回呼び出しが空になるのもそのため。

// recordRow はキャッシュの1行。
type recordRow struct {
	ID          string
	Kind        string
	Title       string
	StartUnix   int64
	EndUnix     int64
	URL         string
	Description string
	Content     string
	Tags        string
	SearchText  string
	RelatedUnix int64
	UpdateUnix  int64
}

const recordColumns = `kyou_id, kind, title, start_unix, end_unix, url, description, content, tags, `

func scanRecordRow(scan func(...any) error) (recordRow, error) {
	var row recordRow
	err := scan(&row.ID, &row.Kind, &row.Title, &row.StartUnix, &row.EndUnix, &row.URL,
		&row.Description, &row.Content, &row.Tags, &row.SearchText, &row.RelatedUnix, &row.UpdateUnix)
	return row, err
}

// QueryKyous は期間で絞った記録を新しい順に返す。
//
// hasWordFilter が偽なら LIMIT を SQL に押し込み、search_text は読まない。
// 真なら LIMIT を押し込まない（単語で絞る前に切ると、後段のフィルタで落ちたぶん取りこぼす）。
func (c *cache) QueryKyous(pluginDir string, start, end *time.Time, limit int, hasWordFilter bool) ([]recordRow, error) {
	if err := c.openDB(pluginDir); err != nil {
		return nil, err
	}
	db := c.conn()
	if db == nil {
		return nil, fmt.Errorf("cache db is not opened")
	}

	searchColumn := `''`
	if hasWordFilter {
		searchColumn = `search_text`
	}
	query := `SELECT ` + recordColumns + searchColumn + `, related_unix, update_unix FROM record WHERE 1 = 1`
	args := []any{}
	if start != nil {
		query += ` AND related_unix >= ?`
		args = append(args, start.Unix())
	}
	if end != nil {
		query += ` AND related_unix <= ?`
		args = append(args, end.Unix())
	}
	query += ` ORDER BY related_unix DESC, kyou_id`
	if !hasWordFilter && limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error at select record: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []recordRow
	for rows.Next() {
		row, err := scanRecordRow(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("error at scan record: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// QueryKyou はIDで1件返す。
func (c *cache) QueryKyou(pluginDir, kyouID string) (recordRow, error) {
	if err := c.openDB(pluginDir); err != nil {
		return recordRow{}, err
	}
	db := c.conn()
	if db == nil {
		return recordRow{}, fmt.Errorf("cache db is not opened")
	}
	row, err := scanRecordRow(db.QueryRow(`SELECT `+recordColumns+`search_text, related_unix, update_unix
FROM record WHERE kyou_id = ?`, kyouID).Scan)
	if err != nil {
		return recordRow{}, fmt.Errorf("error at select record %s: %w", kyouID, err)
	}
	return row, nil
}

// cacheStats は設定画面に出す状態。ファイルは1バイトも開かずに作る。
type cacheStats struct {
	CacheDBPath     string
	FileCount       int
	SessionCount    int
	WorldCount      int
	PlayerCount     int
	EventCount      int
	RecordCount     int
	TargetFileCount int
	BuildState      string
	BuildError      string
	BuildTotal      int
	BuildDone       int
	DirtyCount      int
	DroppedLines    int
	SourceProblems  []string
	LastScan        time.Time
	Err             error
}

// Stats はキャッシュの状態を返す。
//
// GetConfigHTML から呼ぶので、ここで走査してはいけない。
// 5秒の IsAlive を超えるとプロセスが殺される。
func (c *cache) Stats(pluginDir string) cacheStats {
	stats := cacheStats{CacheDBPath: sdk.CacheDBPath(pluginDir)}
	if err := c.openDB(pluginDir); err != nil {
		stats.Err = err
		return stats
	}
	db := c.conn()
	if db == nil {
		stats.Err = fmt.Errorf("cache db is not opened")
		return stats
	}

	_ = db.QueryRow(`SELECT COUNT(*) FROM file_cache`).Scan(&stats.FileCount)
	_ = db.QueryRow(`SELECT COUNT(DISTINCT session_key) FROM file_cache WHERE session_key <> ''`).Scan(&stats.SessionCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM record WHERE kind = ?`, kindWorldTimeIs).Scan(&stats.WorldCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM record WHERE kind = ?`, kindPlayer).Scan(&stats.PlayerCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM record WHERE kind = ?`, kindEvent).Scan(&stats.EventCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM record`).Scan(&stats.RecordCount)
	_ = db.QueryRow(`SELECT COALESCE(SUM(dropped_lines), 0) FROM file_cache`).Scan(&stats.DroppedLines)

	stats.BuildState = c.getMeta("build_state")
	stats.BuildError = c.getMeta("build_error")
	stats.BuildTotal = atoiOrZero(c.getMeta("build_total_files"))
	stats.BuildDone = atoiOrZero(c.getMeta("build_done_files"))
	stats.DirtyCount = atoiOrZero(c.getMeta("dirty_session_count"))
	stats.TargetFileCount = atoiOrZero(c.getMeta("target_file_count"))

	if raw := c.getMeta("source_problems"); raw != "" {
		var problems []string
		if json.Unmarshal([]byte(raw), &problems) == nil {
			stats.SourceProblems = problems
		}
	}
	if unix := atoiOrZero(c.getMeta("last_scan_unix")); unix > 0 {
		stats.LastScan = time.Unix(int64(unix), 0)
	}
	return stats
}

func atoiOrZero(s string) int {
	value, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return value
}
