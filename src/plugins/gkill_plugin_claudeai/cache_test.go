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
	id        string
	sender    string
	text      string
	createdAt string // RFC3339
}
type testConv struct {
	uuid      string
	name      string
	createdAt string // RFC3339
	msgs      []testMsg
}

// writeConversations は conversations.json を dir へ書く。
func writeConversations(t *testing.T, dir string, convs []testConv) {
	t.Helper()
	arr := make([]map[string]any, 0, len(convs))
	for _, cv := range convs {
		msgs := make([]map[string]any, 0, len(cv.msgs))
		for _, m := range cv.msgs {
			msgs = append(msgs, map[string]any{
				"uuid":       m.id,
				"text":       m.text,
				"sender":     m.sender,
				"created_at": m.createdAt,
				"updated_at": m.createdAt,
			})
		}
		arr = append(arr, map[string]any{
			"uuid":          cv.uuid,
			"name":          cv.name,
			"created_at":    cv.createdAt,
			"updated_at":    cv.createdAt,
			"chat_messages": msgs,
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
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_claudeai")
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
func TestClaudeAIBatchCommitPersistsEarlierBatches(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	writeConversations(t, dir, []testConv{
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "hello", createdAt: "2021-01-01T00:00:00Z"}}},
		{uuid: "c2", name: "two", createdAt: "2021-01-02T00:00:00Z", msgs: []testMsg{{id: "m2", sender: "human", text: "world", createdAt: "2021-01-02T00:00:00Z"}}},
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
func TestClaudeAIReadsDoNotTakeBuildMu(t *testing.T) {
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
func TestClaudeAIRebuildOnSourceChange(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	src := expandedSource{Dirs: []string{dir}}

	writeConversations(t, dir, []testConv{
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
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
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
		{uuid: "c2", name: "two", createdAt: "2021-01-02T00:00:00Z", msgs: []testMsg{{id: "m2", sender: "human", text: "beta gamma delta", createdAt: "2021-01-02T00:00:00Z"}}},
	})
	if err := c.build(pluginDir, src); err != nil {
		t.Fatalf("build 3: %v", err)
	}
	if got := countMessages(t, c); got != 2 {
		t.Fatalf("after adding a conv want 2 msgs, got %d", got)
	}

	// 会話を削除(c1 を消す)→ gen 掃除で m1 が落ち、1件に。
	writeConversations(t, dir, []testConv{
		{uuid: "c2", name: "two", createdAt: "2021-01-02T00:00:00Z", msgs: []testMsg{{id: "m2", sender: "human", text: "beta gamma delta", createdAt: "2021-01-02T00:00:00Z"}}},
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
