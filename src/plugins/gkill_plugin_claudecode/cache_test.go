package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		case "main":
			src = mainFixture
		case "agent":
			src = agentFixture
		case "meta":
			src = metaFixture
		case "history":
			src = histFixture
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

// writeSession は合成のメイントランスクリプト(人間1+応答1)を書き出す。
// バッチや差分の検証で2つ目のセッションが要るときに使う。
func writeSession(t *testing.T, dir, name, sessionID, humanText string) {
	t.Helper()
	lines := []string{
		`{"type":"system","subtype":"init","sessionId":"` + sessionID + `","cwd":"C:\\work\\proj2","gitBranch":"dev","timestamp":"2026-02-01T00:00:00Z"}`,
		`{"type":"user","uuid":"` + sessionID + `U1","promptSource":"typed","timestamp":"2026-02-01T01:00:00Z","sessionId":"` + sessionID + `","message":{"role":"user","content":"` + humanText + `"}}`,
		`{"type":"assistant","uuid":"` + sessionID + `A1","timestamp":"2026-02-01T01:00:10Z","sessionId":"` + sessionID + `","message":{"role":"assistant","content":[{"type":"text","text":"応答"}]}}`,
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("write session: %v", err)
	}
}

func newTestCache(t *testing.T) (*pluginCache, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_claudecode")
	if err := os.MkdirAll(pluginDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	c := &pluginCache{}
	t.Cleanup(func() {
		if c.db != nil {
			_ = c.db.Close()
		}
	})
	return c, pluginDir
}

func testSource(sourceDir string) expandedSource {
	return expandSourcePatterns([]string{sourceDir})
}

func mustBuild(t *testing.T, c *pluginCache, pluginDir, sourceDir string) {
	t.Helper()
	if err := c.build(pluginDir, testSource(sourceDir)); err != nil {
		t.Fatalf("build: %v", err)
	}
}

func countRows(t *testing.T, c *pluginCache, query string, args ...any) int {
	t.Helper()
	count := 0
	if err := c.conn().QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return count
}

func TestBuildFromScratch(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "main", "agent", "meta", "history")
	mustBuild(t, c, pluginDir, sourceDir)

	// main / agent / meta の3つは対象、history は kindOther
	if got := countRows(t, c, `SELECT COUNT(*) FROM file_cache WHERE kind != ?`, kindOther); got != 3 {
		t.Errorf("対象ファイル = %d, want 3", got)
	}
	// S1: A0 / U1 / A1(応答まとめ) / U4 / A6 の5件。サブエージェントは畳み込まれる
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache`); got != 5 {
		t.Errorf("message = %d, want 5", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache WHERE message_text = '最初の質問'`); got != 1 {
		t.Errorf("人間の発言が入っていない: %d", got)
	}
	// サブエージェントの本文は親の応答に畳み込まれている
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache WHERE body_json LIKE '%調べました%'`); got != 1 {
		t.Errorf("サブエージェントが畳み込まれていない: %d", got)
	}
	if state := c.getMeta("build_state"); state != "idle" {
		t.Errorf("build_state = %q", state)
	}
}

func TestBatchCommitPersistsEarlierBatches(t *testing.T) {
	// 1バッチ=1トランザクションになっていることを、2バッチ目を失敗させて確かめる。
	// 単一の巨大トランザクションだと、失敗で1バッチ目もロールバックされてしまう。
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "main") // S1
	writeSession(t, sourceDir, "session-s2.jsonl", "S2", "S2の質問")

	old := buildBatchSessions
	buildBatchSessions = 1
	t.Cleanup(func() { buildBatchSessions = old })

	ingestSessionHook = func(sid string) error {
		if sid == "S2" {
			return fmt.Errorf("わざと失敗")
		}
		return nil
	}
	t.Cleanup(func() { ingestSessionHook = nil })

	// セッションはソートしてから流すので、S1 が1バッチ目(commit)、S2 が2バッチ目(失敗)
	if err := c.build(pluginDir, testSource(sourceDir)); err == nil {
		t.Fatal("2バッチ目を失敗させたのにエラーにならない")
	}

	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache WHERE session_id = 'S1'`); got == 0 {
		t.Error("1バッチ目(S1)がコミットされていない。バッチ境界でトランザクションが分かれていない")
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache WHERE session_id = 'S2'`); got != 0 {
		t.Errorf("失敗した2バッチ目(S2)の行が残っている: %d", got)
	}
}

func TestReadDoesNotBlockOnBuildLock(t *testing.T) {
	// 構築ロックを保持したまま読めることを確かめる。
	// 兼用ロックだと GetMessages がここで詰まり、IsAlive(5秒)の期限でプロセスが殺される。
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "main", "agent", "meta")
	mustBuild(t, c, pluginDir, sourceDir)

	c.buildMu.Lock()
	defer c.buildMu.Unlock()

	done := make(chan error, 1)
	go func() {
		_, err := c.GetMessages(pluginDir)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("構築ロック保持中の読み取りが失敗した: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("構築ロック保持中に GetMessages が返らない (buildMu を取っている)")
	}
}

func TestConcurrentReadDuringBuild(t *testing.T) {
	// 構築と読み取りを同じロックで直列化すると、初回構築のあいだ
	// find_kyous が全部詰まって gkill のデッドラインで殺される。
	// WAL + ロック分割の回帰テスト。
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "main", "agent", "meta", "history")
	mustBuild(t, c, pluginDir, sourceDir)

	var wg sync.WaitGroup
	done := make(chan struct{})
	errs := make(chan error, 64)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		for range 5 {
			if err := c.build(pluginDir, testSource(sourceDir)); err != nil {
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
				if _, err := c.GetMessages(pluginDir); err != nil {
					errs <- err
					return
				}
				_ = c.GetStats(pluginDir)
			}
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("構築中の読み取りが失敗した: %v", err)
	}
}

func TestDifferentialUpdatePerSession(t *testing.T) {
	// 変わったセッションだけ作り直し、変わっていないセッションは読み直さない。
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "main") // S1
	writeSession(t, sourceDir, "session-s2.jsonl", "S2", "S2の質問")
	mustBuild(t, c, pluginDir, sourceDir)

	s1Before := countRows(t, c, `SELECT COUNT(*) FROM message_cache WHERE session_id = 'S1'`)
	if s1Before == 0 {
		t.Fatal("S1 の発言が入っていない")
	}

	// S2 の行に印を付ける。再取り込みされたら消える。
	if _, err := c.conn().Exec(`UPDATE message_cache SET message_text = 'SENTINEL' WHERE session_id = 'S2'`); err != nil {
		t.Fatalf("mark: %v", err)
	}

	// S1 の main にだけ人間の発言を追記する
	target := filepath.Join(sourceDir, filepath.Base(mainFixture))
	appended := `{"type":"user","uuid":"U9","promptSource":"typed","timestamp":"2026-01-01T03:00:00Z","sessionId":"S1","message":{"role":"user","content":"追記した質問"}}` + "\n"
	f, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := f.WriteString(appended); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = f.Close()

	mustBuild(t, c, pluginDir, sourceDir)

	// S1 は作り直され、追記した発言が入る
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache WHERE message_text = '追記した質問'`); got != 1 {
		t.Errorf("追記した発言が反映されていない: %d", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache WHERE session_id = 'S1'`); got != s1Before+1 {
		t.Errorf("S1 の件数 = %d, want %d", got, s1Before+1)
	}
	// S2 は触られていない(印が残る)
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache WHERE session_id = 'S2' AND message_text = 'SENTINEL'`); got == 0 {
		t.Error("変わっていない S2 が再取り込みされている(印が消えた)")
	}
}

func TestRemovedFileRemovesKyous(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "main", "agent", "meta")
	mustBuild(t, c, pluginDir, sourceDir)

	// サブエージェントを消すと、親の応答から畳み込みが消える
	if err := os.Remove(filepath.Join(sourceDir, filepath.Base(agentFixture))); err != nil {
		t.Fatalf("remove agent: %v", err)
	}
	mustBuild(t, c, pluginDir, sourceDir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache WHERE body_json LIKE '%調べました%'`); got != 0 {
		t.Errorf("消えたサブエージェントが親に残っている: %d", got)
	}

	// 親(main)を消すと、そのセッションの発言がすべて消える
	if err := os.Remove(filepath.Join(sourceDir, filepath.Base(mainFixture))); err != nil {
		t.Fatalf("remove main: %v", err)
	}
	mustBuild(t, c, pluginDir, sourceDir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache`); got != 0 {
		t.Errorf("消えたセッションの発言が残っている: %d", got)
	}
}

func TestSchemaVersionBumpRebuilds(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "main", "agent", "meta")
	mustBuild(t, c, pluginDir, sourceDir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM message_cache`); got == 0 {
		t.Fatal("最初の構築で発言ができていない")
	}

	if _, err := c.conn().Exec(`INSERT OR REPLACE INTO cache_meta(key, value) VALUES('schema_version', 'old')`); err != nil {
		t.Fatalf("downgrade: %v", err)
	}
	_ = c.db.Close()
	c.db = nil

	fresh := &pluginCache{}
	t.Cleanup(func() {
		if fresh.db != nil {
			_ = fresh.db.Close()
		}
	})
	if err := fresh.openDB(pluginDir); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	if got := countRows(t, fresh, `SELECT COUNT(*) FROM message_cache`); got != 0 {
		t.Errorf("スキーマ世代が違うのに作り直していない: %d件残っている", got)
	}
	if version := fresh.getMeta("schema_version"); version != cacheSchemaVersion {
		t.Errorf("schema_version = %q", version)
	}
}

func TestStatsReportsState(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "main", "agent", "meta", "history")
	mustBuild(t, c, pluginDir, sourceDir)

	stats := c.GetStats(pluginDir)
	if stats.LastScanError != "" {
		t.Fatalf("LastScanError = %q", stats.LastScanError)
	}
	if stats.MessageCount != 5 {
		t.Errorf("MessageCount = %d, want 5", stats.MessageCount)
	}
	if stats.FileCount != 3 {
		t.Errorf("FileCount = %d, want 3", stats.FileCount)
	}
	if stats.BuildState != "idle" {
		t.Errorf("BuildState = %q", stats.BuildState)
	}
	if stats.LastScanUnix == 0 {
		t.Error("LastScanUnix が入っていない")
	}
}
