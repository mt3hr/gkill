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

const selectCommitColumns = `hash, rep_name, committer_unix, tz_offset_sec, author_name, author_email, message, addition, deletion`

func scanCommitRow(scanner interface{ Scan(dest ...any) error }) (commitRow, error) {
	var row commitRow
	err := scanner.Scan(&row.Hash, &row.RepName, &row.CommitterUnix, &row.TZOffsetSec,
		&row.AuthorName, &row.AuthorEmail, &row.Message, &row.Addition, &row.Deletion)
	return row, err
}

// QueryCommits は期間で絞ったコミットをコミッタ日時の降順で返す。
//
// hasWordFilter が要点。偽なら LIMIT を SQL に押し込む。
// 真なら押し込まない —— 単語で絞る前に切ると、後段のフィルタで落ちたぶん取りこぼす。
func (c *cache) QueryCommits(pluginDir string, start, end *time.Time, limit int, hasWordFilter bool) ([]commitRow, error) {
	if err := c.openDB(pluginDir); err != nil {
		return nil, err
	}
	db := c.conn()
	if db == nil {
		return nil, fmt.Errorf("cache db is not opened")
	}

	query := `SELECT ` + selectCommitColumns + ` FROM commit_log WHERE 1 = 1`
	args := []any{}
	if start != nil {
		query += ` AND committer_unix >= ?`
		args = append(args, start.Unix())
	}
	if end != nil {
		query += ` AND committer_unix <= ?`
		args = append(args, end.Unix())
	}
	query += ` ORDER BY committer_unix DESC, hash`
	if !hasWordFilter && limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error at select commit_log: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []commitRow{}
	for rows.Next() {
		row, err := scanCommitRow(rows)
		if err != nil {
			return nil, fmt.Errorf("error at scan commit_log: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// QueryCommit はハッシュで1件返す。
func (c *cache) QueryCommit(pluginDir string, hash string) (commitRow, error) {
	if err := c.openDB(pluginDir); err != nil {
		return commitRow{}, err
	}
	db := c.conn()
	if db == nil {
		return commitRow{}, fmt.Errorf("cache db is not opened")
	}
	row, err := scanCommitRow(db.QueryRow(`SELECT `+selectCommitColumns+` FROM commit_log WHERE hash = ?`, hash))
	if err != nil {
		return commitRow{}, fmt.Errorf("error at select commit %s: %w", hash, err)
	}
	return row, nil
}

// QueryBody は詳細ビュー用の1件（ファイル別の行数と出どころ込み）を返す。一覧では呼ばないこと。
func (c *cache) QueryBody(pluginDir string, hash string) (commitBody, error) {
	row, err := c.QueryCommit(pluginDir, hash)
	if err != nil {
		return commitBody{}, err
	}
	db := c.conn()

	body := commitBody{commitRow: row}
	files, statsError := "", ""
	if err := db.QueryRow(`SELECT file_stats_json, stats_error FROM commit_log WHERE hash = ?`, hash).Scan(&files, &statsError); err != nil {
		return commitBody{}, fmt.Errorf("error at select file stats %s: %w", hash, err)
	}
	if files != "" {
		if err := json.Unmarshal([]byte(files), &body.Files); err != nil {
			return commitBody{}, fmt.Errorf("error at unmarshal file stats %s: %w", hash, err)
		}
	}
	body.StatsError = statsError

	sources, err := db.Query(`SELECT archive_path, git_dir FROM commit_repo WHERE hash = ? ORDER BY archive_path, git_dir`, hash)
	if err != nil {
		return commitBody{}, fmt.Errorf("error at select commit_repo %s: %w", hash, err)
	}
	defer func() { _ = sources.Close() }()
	for sources.Next() {
		var source commitSource
		if err := sources.Scan(&source.ArchivePath, &source.GitDir); err != nil {
			return commitBody{}, fmt.Errorf("error at scan commit_repo %s: %w", hash, err)
		}
		body.Sources = append(body.Sources, source)
	}
	return body, sources.Err()
}

// RepNames は取り込んだコミットが名乗る rep 名（リポジトリ名）の一覧を名前順で返す。
// get_rep_name の rep_names に載る。まだ1件も無ければ空スライス（nil ではない）。
func (c *cache) RepNames(pluginDir string) ([]string, error) {
	if err := c.openDB(pluginDir); err != nil {
		return nil, err
	}
	db := c.conn()
	if db == nil {
		return nil, fmt.Errorf("cache db is not opened")
	}
	rows, err := db.Query(`SELECT DISTINCT rep_name FROM commit_log ORDER BY rep_name`)
	if err != nil {
		return nil, fmt.Errorf("error at select rep names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	names := []string{}
	for rows.Next() {
		name := ""
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("error at scan rep name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// repoRow は設定画面に出すリポジトリ1件。
type repoRow struct {
	ArchivePath string
	GitDir      string
	RepName     string
	CommitCount int
	Note        string
}

// cacheStats は設定画面に出す状態。zip は1つも開かずに作る。
type cacheStats struct {
	CacheDBPath     string
	RepoCount       int
	RepNameCount    int
	CommitCount     int
	StatsErrorCount int
	TargetRepoCount int
	BuildState      string
	BuildError      string
	BuildTotal      int
	BuildDone       int
	SourceProblems  []sourceProblemRow
	Repos           []repoRow
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

	_ = db.QueryRow(`SELECT COUNT(*) FROM repo`).Scan(&stats.RepoCount)
	_ = db.QueryRow(`SELECT COUNT(DISTINCT rep_name) FROM commit_log`).Scan(&stats.RepNameCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM commit_log`).Scan(&stats.CommitCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM commit_log WHERE stats_error != ''`).Scan(&stats.StatsErrorCount)

	stats.BuildState = c.getMeta("build_state")
	stats.BuildError = c.getMeta("build_error")
	stats.BuildTotal = atoiOrZero(c.getMeta("build_total_repos"))
	stats.BuildDone = atoiOrZero(c.getMeta("build_done_repos"))
	stats.TargetRepoCount = atoiOrZero(c.getMeta("target_repo_count"))
	stats.SourceProblems = c.loadSourceProblems()
	if unix := atoiOrZero(c.getMeta("last_scan_unix")); unix > 0 {
		stats.LastScan = time.Unix(int64(unix), 0)
	}

	rows, err := db.Query(`SELECT archive_path, git_dir, rep_name, commit_count, note FROM repo ORDER BY archive_path, git_dir`)
	if err != nil {
		return stats
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var repo repoRow
		if err := rows.Scan(&repo.ArchivePath, &repo.GitDir, &repo.RepName, &repo.CommitCount, &repo.Note); err != nil {
			break
		}
		stats.Repos = append(stats.Repos, repo)
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
