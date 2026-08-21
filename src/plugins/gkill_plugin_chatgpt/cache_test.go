package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testMsg / testConv は合成の会話フィクスチャ。実データは使わない(PII対策)。
type testMsg struct {
	id    string
	role  string
	text  string
	ctime float64
}
type testConv struct {
	id    string
	title string
	ctime float64
	msgs  []testMsg
}

// writeConversations は conversations.json を dir へ書く。
func writeConversations(t *testing.T, dir string, convs []testConv) {
	t.Helper()
	arr := make([]map[string]any, 0, len(convs))
	for _, cv := range convs {
		mapping := map[string]any{}
		for i, m := range cv.msgs {
			nodeID := fmt.Sprintf("%s-node-%d", cv.id, i)
			mapping[nodeID] = map[string]any{
				"id": nodeID,
				"message": map[string]any{
					"id":          m.id,
					"author":      map[string]any{"role": m.role},
					"content":     map[string]any{"content_type": "text", "parts": []any{m.text}},
					"create_time": m.ctime,
				},
			}
		}
		arr = append(arr, map[string]any{
			"id":          cv.id,
			"title":       cv.title,
			"create_time": cv.ctime,
			"mapping":     mapping,
		})
	}
	data, err := json.MarshalIndent(arr, "", " ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conversations.json"), data, 0o600); err != nil {
		t.Fatalf("write conversations.json: %v", err)
	}
}

// newTestCache は空のキャッシュとプラグインフォルダを用意する。
func newTestCache(t *testing.T) (*pluginCache, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_chatgpt")
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

func countMessages(t *testing.T, c *pluginCache) int {
	t.Helper()
	n := 0
	if err := c.conn().QueryRow(`SELECT COUNT(*) FROM msg_cache`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// バッチ永続性: 1バッチ目 commit 後に2つ目を故意に失敗させても、1つ目の行は残る。
// 単一トランザクション同期構築だと全ロールバックで進捗ゼロに戻っていた(M-6)。
func TestChatGPTBatchCommitPersistsEarlierBatches(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	writeConversations(t, dir, []testConv{
		{id: "c1", title: "one", ctime: 1000, msgs: []testMsg{{id: "m1", role: "user", text: "hello", ctime: 1000}}},
		{id: "c2", title: "two", ctime: 2000, msgs: []testMsg{{id: "m2", role: "user", text: "world", ctime: 2000}}},
	})
	src := expandedSource{Dirs: []string{dir}}

	oldBatch := buildBatchConvs
	buildBatchConvs = 1
	defer func() { buildBatchConvs = oldBatch }()
	ingestConvHook = func(convID string) error {
		if convID == "c2" {
			return fmt.Errorf("injected failure on c2")
		}
		return nil
	}
	defer func() { ingestConvHook = nil }()

	if err := c.build(pluginDir, src); err == nil {
		t.Fatal("expected build error from injected failure")
	}

	msgs, err := c.GetMessages(pluginDir)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	seen := map[string]bool{}
	for _, m := range msgs {
		seen[m.MsgID] = true
	}
	if !seen["m1"] {
		t.Error("first batch (m1) should have persisted despite the second batch failing")
	}
	if seen["m2"] {
		t.Error("failed batch (m2) must not be committed")
	}
}

// 読み取りは buildMu を取らない: 構築ロック保持中でも GetMessages は返る。
func TestChatGPTReadsDoNotTakeBuildMu(t *testing.T) {
	c, pluginDir := newTestCache(t)
	if err := c.openDB(pluginDir); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	c.buildMu.Lock()
	defer c.buildMu.Unlock()

	done := make(chan struct{})
	go func() {
		_, _ = c.GetMessages(pluginDir)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("GetMessages blocked on buildMu (reads must not take the build lock)")
	}
}

// 署名変更で作り直し、gen 掃除で消えた会話が落ちる。
func TestChatGPTRebuildOnSourceChange(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	src := expandedSource{Dirs: []string{dir}}

	writeConversations(t, dir, []testConv{
		{id: "c1", title: "one", ctime: 1000, msgs: []testMsg{{id: "m1", role: "user", text: "alpha", ctime: 1000}}},
	})
	if err := c.build(pluginDir, src); err != nil {
		t.Fatalf("build 1: %v", err)
	}
	if got := countMessages(t, c); got != 1 {
		t.Fatalf("after build 1 want 1 msg, got %d", got)
	}

	// 署名が変わらなければ作り直さない(no-op)。
	if err := c.build(pluginDir, src); err != nil {
		t.Fatalf("build 2 (no-op): %v", err)
	}
	if got := countMessages(t, c); got != 1 {
		t.Fatalf("after no-op build want 1 msg, got %d", got)
	}

	// 会話を追加(サイズが変わり署名も変わる)→ 作り直しで2件に。
	writeConversations(t, dir, []testConv{
		{id: "c1", title: "one", ctime: 1000, msgs: []testMsg{{id: "m1", role: "user", text: "alpha", ctime: 1000}}},
		{id: "c2", title: "two", ctime: 2000, msgs: []testMsg{{id: "m2", role: "user", text: "beta gamma delta", ctime: 2000}}},
	})
	if err := c.build(pluginDir, src); err != nil {
		t.Fatalf("build 3: %v", err)
	}
	if got := countMessages(t, c); got != 2 {
		t.Fatalf("after adding a conv want 2 msgs, got %d", got)
	}

	// 会話を削除(c1 を消す)→ gen 掃除で m1 が落ち、1件に。
	writeConversations(t, dir, []testConv{
		{id: "c2", title: "two", ctime: 2000, msgs: []testMsg{{id: "m2", role: "user", text: "beta gamma delta", ctime: 2000}}},
	})
	if err := c.build(pluginDir, src); err != nil {
		t.Fatalf("build 4: %v", err)
	}
	if got := countMessages(t, c); got != 1 {
		t.Fatalf("after removing a conv want 1 msg, got %d", got)
	}
	msgs, _ := c.GetMessages(pluginDir)
	for _, m := range msgs {
		if m.MsgID == "m1" {
			t.Error("removed conversation's message (m1) should have been cleaned up by gen sweep")
		}
	}
}
