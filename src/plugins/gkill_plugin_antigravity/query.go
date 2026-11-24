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

// kyouRow は一覧に必要なぶんだけの1行。body_json は含めない。
type kyouRow struct {
	ID          string
	Title       string
	Project     string
	SearchText  string
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
//
// ids は gkill の FindQuery.IDs（ADR-0318）。nil は指定なし、[] は0件。
func (c *cache) QueryKyous(pluginDir string, start, end *time.Time, limit int, hasWordFilter bool, ids []string) ([]kyouRow, error) {
	if ids != nil && len(ids) == 0 {
		return nil, nil
	}
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
	query := `SELECT kyou_id, title, project, ` + searchColumn + `, related_unix, update_unix
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
	if ids != nil {
		query += ` AND kyou_id IN (` + inPlaceholders(len(ids)) + `)`
		for _, id := range ids {
			args = append(args, id)
		}
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
		if err := rows.Scan(&row.ID, &row.Title, &row.Project, &row.SearchText,
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
	err := db.QueryRow(`SELECT kyou_id, title, project, search_text, related_unix, update_unix
FROM kyou_cache WHERE kyou_id = ?`, kyouID).Scan(&row.ID, &row.Title, &row.Project,
		&row.SearchText, &row.RelatedUnix, &row.UpdateUnix)
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

	body, title, project := "", "", ""
	if err := db.QueryRow(
		`SELECT body_json, title, project FROM kyou_cache WHERE kyou_id = ?`, kyouID,
	).Scan(&body, &title, &project); err != nil {
		return message{}, fmt.Errorf("error at select body %s: %w", kyouID, err)
	}

	var built message
	if err := json.Unmarshal([]byte(body), &built); err != nil {
		return message{}, fmt.Errorf("error at unmarshal body %s: %w", kyouID, err)
	}
	// 会話タイトルとプロジェクト名は body_json に焼かず別カラムで更新しているので、ここで載せる
	built.Title = title
	built.Project = project
	return built, nil
}

// cacheStats は設定画面に出す状態。ファイルは1バイトも開かずに作る。
type cacheStats struct {
	CacheDBPath       string
	FileCount         int
	ConversationCount int
	KyouCount         int
	TargetFileCount   int
	BuildState        string
	BuildError        string
	BuildTotal        int
	BuildDone         int
	DroppedLines      int
	ShadowedCount     int
	SourceProblems    []string
	MissingIDFiles    []string
	LastScan          time.Time
	Err               error
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

	_ = db.QueryRow(`SELECT COUNT(*) FROM file_cache WHERE kind = ?`, kindTranscript).Scan(&stats.FileCount)
	_ = db.QueryRow(`SELECT COUNT(DISTINCT conversation_id) FROM kyou_cache`).Scan(&stats.ConversationCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM kyou_cache`).Scan(&stats.KyouCount)
	_ = db.QueryRow(`SELECT COALESCE(SUM(dropped_lines), 0) FROM file_cache`).Scan(&stats.DroppedLines)

	stats.BuildState = c.getMeta("build_state")
	stats.BuildError = c.getMeta("build_error")
	stats.BuildTotal = atoiOrZero(c.getMeta("build_total_files"))
	stats.BuildDone = atoiOrZero(c.getMeta("build_done_files"))
	stats.TargetFileCount = atoiOrZero(c.getMeta("target_file_count"))
	stats.ShadowedCount = atoiOrZero(c.getMeta("shadowed_count"))
	stats.SourceProblems = c.loadStrings("source_problems")
	stats.MissingIDFiles = c.loadStrings("missing_id_files")

	if unix := atoiOrZero(c.getMeta("last_scan_unix")); unix > 0 {
		stats.LastScan = time.Unix(int64(unix), 0)
	}
	return stats
}

// loadStrings は cache_meta に控えた文字列の一覧を読む。
func (c *cache) loadStrings(key string) []string {
	raw := c.getMeta(key)
	if raw == "" {
		return nil
	}
	var values []string
	if json.Unmarshal([]byte(raw), &values) != nil {
		return nil
	}
	return values
}

func atoiOrZero(s string) int {
	value, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return value
}

// inPlaceholders は SQL の IN 句の "?, ?, ?" を n 個ぶん作る。
func inPlaceholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}
