package main

import (
	"encoding/json"
	"fmt"
	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	"strconv"
	"strings"
	"time"
)

// 読み取りは buildMu を取らない。構築中でも「そこまで取り込めたぶん」を返す。
// これが仕様どおりの挙動で、初回呼び出しが空になるのもそのため。

// kyouRow は一覧に必要なぶんだけの1行。body_json は含めない。
type kyouRow struct {
	ID          string
	Title       string
	SearchText  string
	Originator  string
	RelatedUnix int64
	UpdateUnix  int64
}

// QueryKyous は期間で絞ったKyouを返す。
//
// hasWordFilter が要点。
//   - 偽なら LIMIT を SQL に押し込み、search_text は読まない
//     (検索用テキストは1件で最大512KBあるので、要らないときに読むと丸損)
//   - 真なら LIMIT を押し込まない。単語で絞る前に切ると、
//     後段のフィルタで落ちたぶん取りこぼす
func (c *cache) QueryKyous(pluginDir string, start, end *time.Time, limit int, hasWordFilter bool) ([]kyouRow, error) {
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
	query := `SELECT kyou_id, title, ` + searchColumn + `, originator, related_unix, update_unix
FROM kyou_cache WHERE 1 = 1`
	args := []any{}
	if start != nil {
		query += ` AND related_unix >= ?`
		args = append(args, start.Unix())
	}
	if end != nil {
		query += ` AND related_unix <= ?`
		args = append(args, end.Unix())
	}
	query += ` ORDER BY related_unix DESC`
	if !hasWordFilter && limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error at select kyou_cache: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []kyouRow
	for rows.Next() {
		var row kyouRow
		if err := rows.Scan(&row.ID, &row.Title, &row.SearchText, &row.Originator,
			&row.RelatedUnix, &row.UpdateUnix); err != nil {
			return nil, fmt.Errorf("error at scan kyou_cache: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// QueryKyou はIDで1件返す。
func (c *cache) QueryKyou(pluginDir, kyouID string) (kyouRow, error) {
	if err := c.openDB(pluginDir); err != nil {
		return kyouRow{}, err
	}
	db := c.conn()
	if db == nil {
		return kyouRow{}, fmt.Errorf("cache db is not opened")
	}

	var row kyouRow
	err := db.QueryRow(`SELECT kyou_id, title, search_text, originator, related_unix, update_unix
FROM kyou_cache WHERE kyou_id = ?`, kyouID).Scan(&row.ID, &row.Title, &row.SearchText,
		&row.Originator, &row.RelatedUnix, &row.UpdateUnix)
	if err != nil {
		return kyouRow{}, fmt.Errorf("error at select kyou %s: %w", kyouID, err)
	}
	return row, nil
}

// QueryBody は詳細ビュー用の本体を返す。一覧では絶対に呼ばないこと。
func (c *cache) QueryBody(pluginDir, kyouID string) (message, error) {
	if err := c.openDB(pluginDir); err != nil {
		return message{}, err
	}
	db := c.conn()
	if db == nil {
		return message{}, fmt.Errorf("cache db is not opened")
	}

	body, title := "", ""
	if err := db.QueryRow(
		`SELECT body_json, title FROM kyou_cache WHERE kyou_id = ?`, kyouID,
	).Scan(&body, &title); err != nil {
		return message{}, fmt.Errorf("error at select body %s: %w", kyouID, err)
	}

	var built message
	if err := json.Unmarshal([]byte(body), &built); err != nil {
		return message{}, fmt.Errorf("error at unmarshal body %s: %w", kyouID, err)
	}
	// スレッド名は body_json に焼かず別カラムで更新しているので、ここで載せる
	built.Title = title
	return built, nil
}

// cacheStats は設定画面に出す状態。ファイルは1バイトも開かずに作る。
type cacheStats struct {
	CacheDBPath     string
	FileCount       int
	ThreadCount     int
	SubAgentCount   int
	KyouCount       int
	TargetFileCount int
	BuildState      string
	BuildError      string
	BuildTotal      int
	BuildDone       int
	DirtyCount      int
	DroppedLines    int
	UnknownKinds    string
	RewriteWarning  string
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

	_ = db.QueryRow(`SELECT COUNT(*) FROM file_cache WHERE kind = ?`, kindRollout).Scan(&stats.FileCount)
	_ = db.QueryRow(`SELECT COUNT(DISTINCT thread_id) FROM file_cache WHERE kind = ?`, kindRollout).Scan(&stats.ThreadCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM file_cache WHERE kind = ? AND is_subagent = 1`, kindRollout).Scan(&stats.SubAgentCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM kyou_cache`).Scan(&stats.KyouCount)
	_ = db.QueryRow(`SELECT COALESCE(SUM(dropped_lines), 0) FROM file_cache`).Scan(&stats.DroppedLines)

	stats.BuildState = c.getMeta("build_state")
	stats.BuildError = c.getMeta("build_error")
	stats.BuildTotal = atoiOrZero(c.getMeta("build_total_files"))
	stats.BuildDone = atoiOrZero(c.getMeta("build_done_files"))
	stats.DirtyCount = atoiOrZero(c.getMeta("dirty_thread_count"))
	stats.TargetFileCount = atoiOrZero(c.getMeta("target_file_count"))
	stats.UnknownKinds = c.getMeta("unknown_kinds")
	stats.RewriteWarning = c.getMeta("rewrite_warning")

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
