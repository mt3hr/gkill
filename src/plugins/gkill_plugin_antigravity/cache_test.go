package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// テストで使う合成の会話ID（testdata と同じもの）。
const (
	conversationA = "00000000-0000-7000-8000-000000000001"
	conversationB = "00000000-0000-7000-8000-000000000002"
)

// fixtureTranscript は testdata の生の置き場にある transcript のパスを返す。
func fixtureTranscript(conversationID string) string {
	return filepath.Join("testdata", "brain", conversationID,
		".system_generated", "logs", transcriptFileName)
}

// stageConversations はフィクスチャを書き換え可能な一時フォルダへ同期先の配置で複製する。
//
//	<tmp>/<会話ID>/transcript_full.jsonl
//
// SyncDatas が作る形と同じなので、会話IDがフォルダ名から採れることもここで確かめられる。
func stageConversations(t *testing.T, conversationIDs ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, conversationID := range conversationIDs {
		stageConversationInto(t, dir, conversationID)
	}
	return dir
}

func stageConversationInto(t *testing.T, dir, conversationID string) string {
	t.Helper()
	data, err := os.ReadFile(fixtureTranscript(conversationID))
	if err != nil {
		t.Fatalf("read fixture %s: %v", conversationID, err)
	}
	target := filepath.Join(dir, conversationID)
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(target, transcriptFileName)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

// writeSummariesDB は conversation_summaries.db を合成して書く。
// バイナリを commit しないため、テストのたびに組む（chatgpt / claudeai が ZIP を組むのと同じ）。
func writeSummariesDB(t *testing.T, path string, titles map[string]string, workspaces map[string]string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open summaries: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS conversation_summaries (
  conversation_id text PRIMARY KEY,
  title           text,
  preview         text,
  step_count      integer,
  workspace_uris  text
)`); err != nil {
		t.Fatalf("create summaries: %v", err)
	}
	for conversationID, title := range titles {
		if _, err := db.Exec(
			`INSERT OR REPLACE INTO conversation_summaries (conversation_id, title, workspace_uris) VALUES (?, ?, ?)`,
			conversationID, title, workspaces[conversationID],
		); err != nil {
			t.Fatalf("insert summaries: %v", err)
		}
	}
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

func testConfig(patterns ...string) pluginConfig {
	return pluginConfig{Patterns: patterns}
}

func mustBuild(t *testing.T, c *cache, pluginDir string, patterns ...string) {
	t.Helper()
	if err := c.build(pluginDir, testConfig(patterns...)); err != nil {
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

// zeroIngestedAt は「このあと取り込み直されたか」を見るための印を付ける。
// ingested_unix は秒精度なので、同じ秒に2回走ると差が出ない。
func zeroIngestedAt(t *testing.T, c *cache) {
	t.Helper()
	if _, err := c.conn().Exec(`UPDATE file_cache SET ingested_unix = 0`); err != nil {
		t.Fatalf("update ingested_unix: %v", err)
	}
}

// 1から構築すると、会話ごとに「人間の発言」と「応答」の Kyou が並ぶ。
func TestBuildFromScratch(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageConversations(t, conversationA, conversationB)
	mustBuild(t, c, pluginDir, sourceDir)

	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache`); got != 5 {
		t.Errorf("Kyou数 = %d, want 5（会話Aで4件・会話Bで1件）", got)
	}
	if got := countRows(t, c, `SELECT COUNT(DISTINCT conversation_id) FROM kyou_cache`); got != 2 {
		t.Errorf("会話数 = %d, want 2", got)
	}
	if got := countRows(t, c,
		`SELECT COUNT(*) FROM kyou_cache WHERE conversation_id = ? AND role = ?`,
		conversationA, roleHuman); got != 2 {
		t.Errorf("会話Aの人間の発言 = %d, want 2", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache WHERE kind = ?`, kindTranscript); got != 2 {
		t.Errorf("file_cache の transcript = %d, want 2", got)
	}

	// 会話タイトルは conversation_summaries.db が無いので最初の発言の冒頭で代用する
	title := ""
	if err := c.conn().QueryRow(
		`SELECT title FROM kyou_cache WHERE conversation_id = ? LIMIT 1`, conversationA,
	).Scan(&title); err != nil {
		t.Fatalf("select title: %v", err)
	}
	if title != "テストの記録を作ってください" {
		t.Errorf("代用タイトル = %q", title)
	}
}

// 変わっていないファイルは2周目で読み直さない。
func TestBuildIsIncrementalPerFile(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageConversations(t, conversationA, conversationB)
	mustBuild(t, c, pluginDir, sourceDir)
	zeroIngestedAt(t, c)
	mustBuild(t, c, pluginDir, sourceDir)

	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache WHERE ingested_unix != 0`); got != 0 {
		t.Errorf("取り込み直されたファイル = %d件, want 0件", got)
	}
}

// conversation_summaries.db だけ変わったときは、タイトルを差し替えるだけで
// transcript を1バイトも読み直さない（タイトルは search_text に焼いていない）。
func TestSummariesChangeRefreshesTitleWithoutReingest(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageConversations(t, conversationA)
	summariesPath := filepath.Join(sourceDir, summariesFileName)
	writeSummariesDB(t, summariesPath,
		map[string]string{conversationA: "最初のタイトル"},
		map[string]string{conversationA: `["file:///c:/example/repo-alpha"]`})
	mustBuild(t, c, pluginDir, sourceDir)

	title, project := "", ""
	if err := c.conn().QueryRow(
		`SELECT title, project FROM kyou_cache WHERE conversation_id = ? LIMIT 1`, conversationA,
	).Scan(&title, &project); err != nil {
		t.Fatalf("select: %v", err)
	}
	if title != "最初のタイトル" || project != "repo-alpha" {
		t.Fatalf("title=%q project=%q", title, project)
	}

	zeroIngestedAt(t, c)
	writeSummariesDB(t, summariesPath,
		map[string]string{conversationA: "あとで付けたタイトル"},
		map[string]string{conversationA: `["file:///c:/example/repo-alpha"]`})
	mustBuild(t, c, pluginDir, sourceDir)

	if err := c.conn().QueryRow(
		`SELECT title FROM kyou_cache WHERE conversation_id = ? LIMIT 1`, conversationA,
	).Scan(&title); err != nil {
		t.Fatalf("select title: %v", err)
	}
	if title != "あとで付けたタイトル" {
		t.Errorf("タイトルが更新されていない: %q", title)
	}
	if got := countRows(t, c,
		`SELECT COUNT(*) FROM file_cache WHERE kind = ? AND ingested_unix != 0`, kindTranscript); got != 0 {
		t.Errorf("transcript が取り込み直された = %d件, want 0件", got)
	}
}

// ファイルが消えたら、その会話の Kyou も消える。
func TestRemovedFileDropsKyou(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageConversations(t, conversationA, conversationB)
	mustBuild(t, c, pluginDir, sourceDir)

	if err := os.RemoveAll(filepath.Join(sourceDir, conversationB)); err != nil {
		t.Fatalf("remove: %v", err)
	}
	mustBuild(t, c, pluginDir, sourceDir)

	if got := countRows(t, c,
		`SELECT COUNT(*) FROM kyou_cache WHERE conversation_id = ?`, conversationB); got != 0 {
		t.Errorf("消えた会話の Kyou = %d件, want 0件", got)
	}
	if got := countRows(t, c,
		`SELECT COUNT(*) FROM kyou_cache WHERE conversation_id = ?`, conversationA); got == 0 {
		t.Error("残っている会話の Kyou まで消えている")
	}
}

// スキーマ版を上げたらキャッシュを作り直す。
func TestSchemaVersionChangeRebuilds(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageConversations(t, conversationA)
	mustBuild(t, c, pluginDir, sourceDir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache`); got == 0 {
		t.Fatal("1周目で0件")
	}

	c.setMeta("schema_version", "0")
	if _, err := c.conn().Exec(`DELETE FROM file_cache`); err != nil {
		t.Fatalf("delete file_cache: %v", err)
	}
	if err := c.db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	c.db = nil

	if err := c.openDB(pluginDir); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache`); got != 0 {
		t.Errorf("作り直されていない: kyou_cache = %d件", got)
	}
	mustBuild(t, c, pluginDir, sourceDir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache`); got == 0 {
		t.Error("作り直したあとに取り込めていない")
	}
}

// 構築中でも読み取りは詰まらない（読み取りは mu も buildMu も取らない）。
func TestConcurrentReadDuringBuild(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	for i := range 12 {
		conversationID := fmt.Sprintf("00000000-0000-7000-8000-%012d", i+1)
		data, err := os.ReadFile(fixtureTranscript(conversationA))
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		target := filepath.Join(dir, conversationID)
		if err := os.MkdirAll(target, os.ModePerm); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(target, transcriptFileName), data, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if err := c.openDB(pluginDir); err != nil {
		t.Fatalf("openDB: %v", err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := c.build(pluginDir, testConfig(dir)); err != nil {
			t.Errorf("build: %v", err)
		}
	})
	for range 30 {
		if _, err := c.QueryKyous(pluginDir, nil, nil, 0, false, nil); err != nil {
			t.Fatalf("構築中の読み取りが失敗した: %v", err)
		}
	}
	wg.Wait()

	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache`); got != 12*4 {
		t.Errorf("Kyou数 = %d, want %d", got, 12*4)
	}
}

// 単語で絞るときは SQL の LIMIT を押し込まない（絞る前に切ると取りこぼす）。
func TestQueryKyousDoesNotPushLimitWhenWordFilter(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageConversations(t, conversationA, conversationB)
	mustBuild(t, c, pluginDir, sourceDir)

	withoutWords, err := c.QueryKyous(pluginDir, nil, nil, 2, false, nil)
	if err != nil {
		t.Fatalf("QueryKyous: %v", err)
	}
	if len(withoutWords) != 2 {
		t.Errorf("単語指定なしの LIMIT = %d件, want 2件", len(withoutWords))
	}
	if withoutWords[0].SearchText != "" {
		t.Error("単語指定が無いのに search_text を読んでいる")
	}

	withWords, err := c.QueryKyous(pluginDir, nil, nil, 2, true, nil)
	if err != nil {
		t.Fatalf("QueryKyous: %v", err)
	}
	if len(withWords) != 5 {
		t.Errorf("単語指定ありで LIMIT を押し込んでいる = %d件, want 5件", len(withWords))
	}
	if withWords[0].SearchText == "" {
		t.Error("単語指定があるのに search_text が空")
	}
}
