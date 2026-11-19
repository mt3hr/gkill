package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// stageLog はフィクスチャを書き換え可能な一時ディレクトリへ複製する。name を空にすると元の名前のまま。
func stageLog(t *testing.T, dir, src, name string) string {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if name == "" {
		name = filepath.Base(src)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func newTestCache(t *testing.T) (*cache, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)
	pluginDir := filepath.Join(home, "plugins", "testuser", appName)
	if err := os.MkdirAll(pluginDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	c := &cache{}
	t.Cleanup(func() {
		if c.db != nil {
			_ = c.db.Close()
		}
	})
	return c, pluginDir
}

func mustBuild(t *testing.T, c *cache, pluginDir string, patterns ...string) {
	t.Helper()
	if err := c.build(pluginDir, pluginConfig{Patterns: patterns}); err != nil {
		t.Fatalf("build: %v", err)
	}
}

func countRows(t *testing.T, c *cache, query string, args ...any) int {
	t.Helper()
	count := 0
	if err := c.conn().QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return count
}

// recordCountOf は2つのフィクスチャから作られる記録の数（テストデータ由来の期待値）。
// 正常終了: ワールド2・URLog2・人4・アバター1・動画1 = 10 / 落ちた: ワールド1・URLog1・人1 = 3
const recordCountOfFixtures = 13

func TestBuildFromScratch(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	stageLog(t, dir, normalFixture(), "")
	stageLog(t, dir, crashFixture(), "")
	// 同じフォルダにある output_log_*.txt ではないファイルは対象外
	stageLog(t, dir, filepath.Join("testdata", "fixtures", "not_an_output_log.txt"), "")
	mustBuild(t, c, pluginDir, dir)

	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache`); got != 2 {
		t.Errorf("file_cache = %d, want 2（not_an_output_log.txt は対象外）", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM record`); got != recordCountOfFixtures {
		t.Errorf("record = %d, want %d", got, recordCountOfFixtures)
	}
	stats := c.Stats(pluginDir)
	if stats.SessionCount != 2 || stats.WorldCount != 3 || stats.PlayerCount != 5 || stats.EventCount != 1 || stats.VideoCount != 1 {
		t.Errorf("stats = %+v", stats)
	}
	if stats.BuildState != "idle" {
		t.Errorf("build_state = %q", stats.BuildState)
	}
}

// 同じ起動のログが2か所にある（元のファイルと、プレイ中に運んだ短いコピー）とき、
// 大きいほうから1回ぶんだけ記録を作る。
func TestDuplicateSessionUsesLargestFile(t *testing.T) {
	c, pluginDir := newTestCache(t)
	original := t.TempDir()
	copied := t.TempDir()
	stageLog(t, original, normalFixture(), "")

	data, err := os.ReadFile(normalFixture())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.SplitAfter(string(data), "\n")
	shortPath := filepath.Join(copied, filepath.Base(normalFixture()))
	if err := os.WriteFile(shortPath, []byte(strings.Join(lines[:8], "")), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	mustBuild(t, c, pluginDir, original, copied)
	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache`); got != 2 {
		t.Errorf("file_cache = %d, want 2", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM record`); got != 10 {
		t.Errorf("record = %d, want 完全な版の10件だけ", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM record WHERE kind = ? AND end_unix = ?`,
		kindWorldTimeIs, testTime(t, "2024.01.01 11:00:00").Unix()); got != 1 {
		t.Errorf("完全な版の終了時刻の滞在 = %d件, want 1件", got)
	}

	// 大きいほうを消すと、残った短い版から作り直す
	if err := os.Remove(filepath.Join(original, filepath.Base(normalFixture()))); err != nil {
		t.Fatalf("remove: %v", err)
	}
	mustBuild(t, c, pluginDir, original, copied)
	short := buildRecords(mustParse(t, shortPath))
	if got := countRows(t, c, `SELECT COUNT(*) FROM record`); got != len(short) {
		t.Errorf("record = %d, want 短い版の %d件", got, len(short))
	}
}

func mustParse(t *testing.T, path string) parsedLog {
	t.Helper()
	parsed, err := parseLogFile(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return parsed
}

// 変わっていないファイルは読み直さず、追記されたファイルだけを読み直す。
func TestBuildIsIncrementalPerFile(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	normalPath := stageLog(t, dir, normalFixture(), "")
	crashPath := stageLog(t, dir, crashFixture(), "")
	mustBuild(t, c, pluginDir, dir)

	if _, err := c.conn().Exec(`UPDATE file_cache SET ingested_unix = 0`); err != nil {
		t.Fatalf("mark: %v", err)
	}
	mustBuild(t, c, pluginDir, dir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache WHERE ingested_unix <> 0`); got != 0 {
		t.Errorf("変わっていないのに読み直したファイル = %d", got)
	}

	// プレイが続いてログが伸びた
	file, err := os.OpenFile(crashPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := file.WriteString("2024.01.02 11:00:00 Debug      -  [Behaviour] OnLeftRoom\n"); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = file.Close()
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(crashPath, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	mustBuild(t, c, pluginDir, dir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache WHERE ingested_unix <> 0 AND path = ?`, normalPath); got != 0 {
		t.Error("追記していないファイルまで読み直した")
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM record WHERE kind = ? AND end_unix = ?`,
		kindWorldTimeIs, testTime(t, "2024.01.02 11:00:00").Unix()); got != 1 {
		t.Errorf("伸びたログで滞在の終了が伸びていない")
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM record`); got != recordCountOfFixtures {
		t.Errorf("record = %d, want %d（記録が二重にならない）", got, recordCountOfFixtures)
	}
}

func TestRemovedFileRemovesRecords(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	stageLog(t, dir, normalFixture(), "")
	crashPath := stageLog(t, dir, crashFixture(), "")
	mustBuild(t, c, pluginDir, dir)

	if err := os.Remove(crashPath); err != nil {
		t.Fatalf("remove: %v", err)
	}
	mustBuild(t, c, pluginDir, dir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM record`); got != 10 {
		t.Errorf("record = %d, want 消したファイルのぶんが消えて10件", got)
	}
}

func TestSchemaVersionBumpRebuilds(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	stageLog(t, dir, normalFixture(), "")
	mustBuild(t, c, pluginDir, dir)

	c.setMeta("schema_version", "0")
	_ = c.db.Close()
	c.db = nil
	if err := c.openDB(pluginDir); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache`); got != 0 {
		t.Errorf("世代が変わったのに file_cache が残っている: %d", got)
	}
	mustBuild(t, c, pluginDir, dir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM record`); got != 10 {
		t.Errorf("作り直し後の record = %d", got)
	}
}

func TestQueryKyousLimitAndPeriod(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	stageLog(t, dir, normalFixture(), "")
	stageLog(t, dir, crashFixture(), "")
	mustBuild(t, c, pluginDir, dir)

	rows, err := c.QueryKyous(pluginDir, kyouQuery{limit: 3})
	if err != nil || len(rows) != 3 {
		t.Fatalf("LIMIT 3 = %d件, %v", len(rows), err)
	}
	if rows[0].SearchText != "" {
		t.Error("単語が無いのに search_text を読んでいる")
	}
	start := testTime(t, "2024.01.02 00:00:00")
	rows, err = c.QueryKyous(pluginDir, kyouQuery{start: &start, hasWordFilter: true})
	if err != nil || len(rows) != 3 {
		t.Fatalf("2日目以降 = %d件, %v, want 落ちた起動の3件", len(rows), err)
	}
	if rows[0].SearchText == "" {
		t.Error("単語があるのに search_text を読んでいない")
	}
	row, err := c.QueryKyou(pluginDir, rows[0].ID)
	if err != nil || row.ID != rows[0].ID {
		t.Errorf("QueryKyou = %+v, %v", row, err)
	}
}

// 構築中でも読み取りは待たされない（読み取りは buildMu を取らない）。
func TestConcurrentReadDuringBuild(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	stageLog(t, dir, normalFixture(), "")
	mustBuild(t, c, pluginDir, dir)

	c.buildMu.Lock()
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		_, _ = c.QueryKyous(pluginDir, kyouQuery{})
		_ = c.Stats(pluginDir)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("構築の lock を持っている間に読み取りが詰まった")
	}
	c.buildMu.Unlock()
	wg.Wait()
}

func TestBuildReportsMissingPattern(t *testing.T) {
	c, pluginDir := newTestCache(t)
	missing := filepath.Join(t.TempDir(), "nothing_here_*")
	mustBuild(t, c, pluginDir, missing)
	if problems := c.Stats(pluginDir).SourceProblems; len(problems) != 1 {
		t.Errorf("SourceProblems = %v, want 1件", problems)
	}
}
