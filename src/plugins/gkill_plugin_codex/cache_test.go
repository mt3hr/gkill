package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// stageSources はフィクスチャを書き換え可能な一時ディレクトリへ複製する。
func stageSources(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		var src string
		switch name {
		case "parent":
			src = parentFixture()
		case "sub":
			src = subAgentFixture()
		case "old":
			src = oldFormatFixture()
		case "index":
			src = filepath.Join("testdata", "session_index.jsonl")
		default:
			t.Fatalf("知らないフィクスチャ: %s", name)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(src)), data, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return dir
}

func newTestCache(t *testing.T) (*cache, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_codex")
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

func testConfig(sourceDir string) pluginConfig {
	return pluginConfig{Patterns: []string{sourceDir}, SubagentMode: subagentModeFold, ScanWorkers: 2}
}

func mustBuild(t *testing.T, c *cache, pluginDir, sourceDir string) {
	t.Helper()
	if err := c.build(pluginDir, testConfig(sourceDir)); err != nil {
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
		t.Fatalf("mark: %v", err)
	}
}

func ingestedPaths(t *testing.T, c *cache) map[string]bool {
	t.Helper()
	rows, err := c.conn().Query(`SELECT path, ingested_unix FROM file_cache`)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer func() { _ = rows.Close() }()
	touched := map[string]bool{}
	for rows.Next() {
		path := ""
		unix := int64(0)
		if err := rows.Scan(&path, &unix); err != nil {
			t.Fatalf("scan: %v", err)
		}
		touched[filepath.Base(path)] = unix != 0
	}
	return touched
}

func TestBuildFromScratch(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "sub", "old", "index")
	mustBuild(t, c, pluginDir, sourceDir)

	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache WHERE kind = 'rollout'`); got != 3 {
		t.Errorf("rollout = %d, want 3", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache WHERE is_subagent = 1`); got != 1 {
		t.Errorf("subagent = %d, want 1", got)
	}
	// 親スレッド4件 + 古い版2件。サブエージェントは親へ畳み込まれるのでKyouにならない
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache`); got != 6 {
		t.Errorf("kyou = %d, want 6", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache WHERE thread_id = ?`, subThreadID); got != 0 {
		t.Errorf("サブエージェントがKyouになっている: %d", got)
	}
	// session_index.jsonl のスレッド名が載る
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache WHERE title = 'バグ修正スレッド'`); got != 4 {
		t.Errorf("スレッド名が付いたKyou = %d, want 4", got)
	}
	if got := countRows(t, c, `SELECT COALESCE(SUM(dropped_lines), 0) FROM file_cache`); got != 0 {
		t.Errorf("dropped = %d, want 0", got)
	}
	if state := c.getMeta("build_state"); state != "idle" {
		t.Errorf("build_state = %q", state)
	}
}

func TestBuildIsIncrementalPerFile(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "sub", "old", "index")
	mustBuild(t, c, pluginDir, sourceDir)
	zeroIngestedAt(t, c)

	// 古い版のログにだけ追記する
	target := filepath.Join(sourceDir, filepath.Base(oldFormatFixture()))
	appended := `{"timestamp":"2026-01-01T00:00:03.000Z","type":"event_msg","payload":{"type":"user_message","message":"追記した発言","images":[],"local_images":[],"text_elements":[]}}` + "\n"
	file, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := file.WriteString(appended); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = file.Close()

	mustBuild(t, c, pluginDir, sourceDir)

	touched := ingestedPaths(t, c)
	for name, wasTouched := range touched {
		want := name == filepath.Base(oldFormatFixture())
		if wasTouched != want {
			t.Errorf("%s の取り込み = %v, want %v (変わっていないファイルは二度と開かない)", name, wasTouched, want)
		}
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache WHERE message_text = '追記した発言'`); got != 1 {
		t.Errorf("追記した発言がKyouになっていない: %d", got)
	}
}

func TestSessionIndexChangeDoesNotReparseRollouts(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "sub", "old", "index")
	mustBuild(t, c, pluginDir, sourceDir)
	zeroIngestedAt(t, c)

	// スレッド名だけ書き換える。実データでも名前が付くたびに書き換わるので、
	// これでロールアウトを読み直していたら毎回フル再構築になる。
	indexPath := filepath.Join(sourceDir, "session_index.jsonl")
	renamed := `{"id":"` + parentThreadID + `","thread_name":"名前を変えた","updated_at":"2026-01-03T00:00:00.0000000Z"}` + "\n"
	if err := os.WriteFile(indexPath, []byte(renamed), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	mustBuild(t, c, pluginDir, sourceDir)

	for name, wasTouched := range ingestedPaths(t, c) {
		if name == "session_index.jsonl" {
			continue
		}
		if wasTouched {
			t.Errorf("%s を読み直している(スレッド名の更新だけのはず)", name)
		}
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache WHERE title = '名前を変えた'`); got != 4 {
		t.Errorf("新しいスレッド名が反映されていない: %d", got)
	}
}

func TestBuildDirtiesParentWhenSubAgentAppears(t *testing.T) {
	// 子が後から現れたとき、親を畳み直さないとサブエージェントが出ない。
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "index")
	mustBuild(t, c, pluginDir, sourceDir)

	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache WHERE body_json LIKE '%Singer%'`); got != 0 {
		t.Fatalf("まだ子が居ないのに畳み込まれている: %d", got)
	}

	data, err := os.ReadFile(subAgentFixture())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, filepath.Base(subAgentFixture())), data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	mustBuild(t, c, pluginDir, sourceDir)

	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache WHERE body_json LIKE '%Singer%'`); got != 1 {
		t.Errorf("子が現れたのに親が畳み直されていない: %d", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache WHERE thread_id = ?`, subThreadID); got != 0 {
		t.Errorf("子が独立したKyouになっている: %d", got)
	}
}

func TestRemovedFileRemovesKyousAndRefoldsParent(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "sub", "old", "index")
	mustBuild(t, c, pluginDir, sourceDir)

	// 子を消すと、親から畳み込みが消える
	if err := os.Remove(filepath.Join(sourceDir, filepath.Base(subAgentFixture()))); err != nil {
		t.Fatalf("remove: %v", err)
	}
	mustBuild(t, c, pluginDir, sourceDir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache WHERE body_json LIKE '%Singer%'`); got != 0 {
		t.Errorf("消えた子が親に残っている: %d", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM thread_item WHERE thread_id = ?`, subThreadID); got != 0 {
		t.Errorf("消えたファイルの要素が残っている: %d", got)
	}

	// 親を消すと、そのスレッドのKyouが消える
	if err := os.Remove(filepath.Join(sourceDir, filepath.Base(parentFixture()))); err != nil {
		t.Fatalf("remove: %v", err)
	}
	mustBuild(t, c, pluginDir, sourceDir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache WHERE thread_id = ?`, parentThreadID); got != 0 {
		t.Errorf("消えた親のKyouが残っている: %d", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache`); got != 2 {
		t.Errorf("残るのは古い版の2件だけのはず: %d", got)
	}
}

func TestSchemaVersionBumpRebuilds(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "index")
	mustBuild(t, c, pluginDir, sourceDir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM kyou_cache`); got == 0 {
		t.Fatal("最初の構築でKyouができていない")
	}

	if _, err := c.conn().Exec(`INSERT OR REPLACE INTO cache_meta (key, value) VALUES ('schema_version', 'old')`); err != nil {
		t.Fatalf("downgrade: %v", err)
	}
	_ = c.db.Close()
	c.db = nil

	fresh := &cache{}
	t.Cleanup(func() {
		if fresh.db != nil {
			_ = fresh.db.Close()
		}
	})
	if err := fresh.openDB(pluginDir); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	if got := countRows(t, fresh, `SELECT COUNT(*) FROM kyou_cache`); got != 0 {
		t.Errorf("スキーマ世代が違うのに作り直していない: %d件残っている", got)
	}
	if version := fresh.getMeta("schema_version"); version != cacheSchemaVersion {
		t.Errorf("schema_version = %q", version)
	}
}

func TestRewriteWarningOnShrink(t *testing.T) {
	// 追記のみの前提が破れると ordinal がずれて KyouID が変わる。
	// 自動修復はしないが、気づけるように印を残す。
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "index")
	mustBuild(t, c, pluginDir, sourceDir)
	if warning := c.getMeta("rewrite_warning"); warning != "" {
		t.Fatalf("最初から警告が出ている: %q", warning)
	}

	target := filepath.Join(sourceDir, filepath.Base(parentFixture()))
	shrunk := `{"timestamp":"2026-01-02T01:00:00.000Z","type":"session_meta","payload":{"id":"` + parentThreadID + `","thread_source":"user","cwd":"c:\\work\\myproj"}}` + "\n"
	if err := os.WriteFile(target, []byte(shrunk), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	mustBuild(t, c, pluginDir, sourceDir)

	if warning := c.getMeta("rewrite_warning"); warning == "" {
		t.Error("ファイルが小さくなったのに警告が出ていない")
	}
}

func TestQueryKyousLimitWithWordFilter(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "sub", "old", "index")
	mustBuild(t, c, pluginDir, sourceDir)

	// 単語で絞るときは LIMIT を SQL に押し込まない。
	// 絞る前に切ると、後段のフィルタで落ちたぶん取りこぼす。
	rows, err := c.QueryKyous(pluginDir, nil, nil, 1, true)
	if err != nil {
		t.Fatalf("QueryKyous: %v", err)
	}
	if len(rows) != 6 {
		t.Errorf("単語指定ありで %d 件。LIMIT を押し込んではいけない", len(rows))
	}
	if rows[0].SearchText == "" {
		t.Error("単語指定ありなのに search_text を読んでいない")
	}

	// 単語指定が無いときは LIMIT を押し込み、search_text は読まない
	// (検索用テキストは1件で最大512KBある)
	rows, err = c.QueryKyous(pluginDir, nil, nil, 2, false)
	if err != nil {
		t.Fatalf("QueryKyous: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("単語指定なしで %d 件。LIMIT が効いていない", len(rows))
	}
	for _, row := range rows {
		if row.SearchText != "" {
			t.Error("単語指定が無いのに search_text を読んでいる")
		}
	}

	// 期間で絞れる
	end := time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)
	rows, err = c.QueryKyous(pluginDir, nil, &end, 0, false)
	if err != nil {
		t.Fatalf("QueryKyous: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("期間で絞ると %d 件, want 2", len(rows))
	}
}

func TestQueryBodyAndKyou(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "sub", "index")
	mustBuild(t, c, pluginDir, sourceDir)

	id := kyouIDOf(parentThreadID, roleAssistant, 1)
	row, err := c.QueryKyou(pluginDir, id)
	if err != nil {
		t.Fatalf("QueryKyou: %v", err)
	}
	if row.ID != id {
		t.Errorf("ID = %q", row.ID)
	}

	built, err := c.QueryBody(pluginDir, id)
	if err != nil {
		t.Fatalf("QueryBody: %v", err)
	}
	if built.Title != "バグ修正スレッド" {
		t.Errorf("Title = %q (body_json に焼かず別カラムから載せる)", built.Title)
	}
	if len(built.Items) == 0 {
		t.Error("Items が空")
	}

	if _, err := c.QueryKyou(pluginDir, "居ないID"); err == nil {
		t.Error("存在しないIDでエラーにならない")
	}
}

func TestConcurrentReadDuringBuild(t *testing.T) {
	// 構築と読み取りを同じロックで直列化すると、初回構築のあいだ
	// find_kyous が全部詰まって gkill のデッドラインで殺される。
	// WAL + ロック分割の回帰テスト。
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "sub", "old", "index")
	mustBuild(t, c, pluginDir, sourceDir)

	var wg sync.WaitGroup
	done := make(chan struct{})
	errs := make(chan error, 64)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		for range 5 {
			if err := c.build(pluginDir, testConfig(sourceDir)); err != nil {
				errs <- err
				return
			}
		}
	}()

	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				if _, err := c.QueryKyous(pluginDir, nil, nil, 0, true); err != nil {
					errs <- err
					return
				}
				if _, err := c.QueryKyou(pluginDir, kyouIDOf(parentThreadID, roleHuman, 0)); err != nil {
					errs <- err
					return
				}
				_ = c.Stats(pluginDir)
			}
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("構築中の読み取りが失敗した: %v", err)
	}
}

func TestStatsReportsProgressWithoutScanning(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "sub", "index")
	mustBuild(t, c, pluginDir, sourceDir)

	stats := c.Stats(pluginDir)
	if stats.Err != nil {
		t.Fatalf("Stats: %v", stats.Err)
	}
	if stats.KyouCount != 4 || stats.FileCount != 2 || stats.SubAgentCount != 1 {
		t.Errorf("stats = %+v", stats)
	}
	if stats.BuildState != "idle" {
		t.Errorf("BuildState = %q", stats.BuildState)
	}
	if stats.LastScan.IsZero() {
		t.Error("LastScan が入っていない")
	}
	if stats.CacheDBPath == "" {
		t.Error("CacheDBPath が空")
	}
}

func TestBuildReportsMissingPattern(t *testing.T) {
	c, pluginDir := newTestCache(t)
	missing := filepath.Join(t.TempDir(), "居ないフォルダ")
	if err := c.build(pluginDir, pluginConfig{Patterns: []string{missing}, SubagentMode: subagentModeFold}); err != nil {
		t.Fatalf("build: %v", err)
	}
	stats := c.Stats(pluginDir)
	if len(stats.SourceProblems) != 1 {
		t.Errorf("SourceProblems = %v", stats.SourceProblems)
	}
}
