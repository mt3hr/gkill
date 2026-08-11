package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// dailyMetricColumns は daily_metric から読む列。Scan の順序と一致させること。
const dailyMetricColumns = `kyou_id, metric_key, date_local, title, num_value, unit,
 sample_count, min_value, max_value, devices, source_paths, hour_sums, hour_counts,
 search_text, related_unix, update_unix`

// QueryDailyMetrics は期間に含まれる集計結果を返す。
//
// 期間の絞り込みはSQLに押し込む。ここをGoでやると、
// 期間指定なしの検索で全件（2万件以上）を毎回メモリに載せることになる。
func (c *cache) QueryDailyMetrics(pluginDir string, config pluginConfig, startUnix *int64, endUnix *int64, limit int) ([]dailyMetric, error) {
	// 構築を待たない。取り込めたぶんだけを返す
	// （待つと初回構築のあいだ検索が全部詰まり、プロセスが殺される）。
	if err := c.openDB(pluginDir); err != nil {
		return nil, err
	}

	query := `SELECT ` + dailyMetricColumns + ` FROM daily_metric`
	conditions := []string{}
	args := []any{}
	if startUnix != nil {
		conditions = append(conditions, "related_unix >= ?")
		args = append(args, *startUnix)
	}
	if endUnix != nil {
		conditions = append(conditions, "related_unix <= ?")
		args = append(args, *endUnix)
	}
	if enabled := config.enabledMetrics(); len(enabled) != 0 {
		placeholders := make([]string, 0, len(enabled))
		for key := range enabled {
			placeholders = append(placeholders, "?")
			args = append(args, key)
		}
		conditions = append(conditions, "metric_key IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(conditions) != 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY related_unix DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error at query daily metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	metrics := []dailyMetric{}
	for rows.Next() {
		metric, err := scanDailyMetric(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, rows.Err()
}

// QueryDailyMetric は1件を返す。無ければ (nil, nil)。
func (c *cache) QueryDailyMetric(pluginDir string, kyouID string) (*dailyMetric, error) {
	// 構築を待たない（QueryDailyMetrics と同じ理由）
	if err := c.openDB(pluginDir); err != nil {
		return nil, err
	}

	rows, err := c.db.Query(`SELECT `+dailyMetricColumns+` FROM daily_metric WHERE kyou_id = ?`, kyouID)
	if err != nil {
		return nil, fmt.Errorf("error at query daily metric: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, rows.Err()
	}
	metric, err := scanDailyMetric(rows)
	if err != nil {
		return nil, err
	}
	return &metric, nil
}

func scanDailyMetric(rows *sql.Rows) (dailyMetric, error) {
	var metric dailyMetric
	err := rows.Scan(
		&metric.KyouID, &metric.MetricKey, &metric.DateLocal, &metric.Title,
		&metric.NumValue, &metric.Unit, &metric.SampleCount,
		&metric.MinValue, &metric.MaxValue, &metric.Devices, &metric.SourcePaths,
		&metric.HourSums, &metric.HourCounts, &metric.SearchText,
		&metric.RelatedUnix, &metric.UpdateUnix)
	if err != nil {
		return dailyMetric{}, fmt.Errorf("error at scan daily metric: %w", err)
	}
	return metric, nil
}

// Stats は設定画面に出す統計を返す。
func (c *cache) Stats(pluginDir string, config pluginConfig) cacheStats {
	// 構築を待たない。進捗を出すのが仕事なので、むしろ構築中こそ返る必要がある

	stats := cacheStats{Timezone: config.Timezone}
	if err := c.openDB(pluginDir); err != nil {
		stats.BuildError = err.Error()
		return stats
	}

	_ = c.db.QueryRow(`SELECT COUNT(*) FROM file_cache`).Scan(&stats.ScannedFileCount)
	_ = c.db.QueryRow(`SELECT COUNT(DISTINCT date_local) FROM daily_metric`).Scan(&stats.DayCount)
	_ = c.db.QueryRow(`SELECT COUNT(DISTINCT metric_key) FROM daily_metric`).Scan(&stats.MetricCount)
	_ = c.db.QueryRow(`SELECT COUNT(*) FROM daily_metric`).Scan(&stats.KyouCount)

	stats.TargetFileCount, _ = strconv.Atoi(c.meta("target_file_count"))
	stats.Exports = c.loadExports()
	stats.Problems = c.loadSourceProblems()

	stats.BuildState = c.meta("build_state")
	stats.BuildError = c.meta("build_error")
	stats.BuildDoneFiles, _ = strconv.Atoi(c.meta("build_done_files"))
	stats.BuildTotalFiles, _ = strconv.Atoi(c.meta("build_total_files"))
	stats.LastScanUnix, _ = strconv.ParseInt(c.meta("last_scan_unix"), 10, 64)
	return stats
}

// formatUnix は表示用に時刻を整える。0なら空。
func formatUnix(unix int64) string {
	if unix == 0 {
		return ""
	}
	return time.Unix(unix, 0).Local().Format("2006-01-02 15:04:05")
}
