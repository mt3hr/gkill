package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// testMsg / testConv は合成の会話フィクスチャ。実データは使わない(PII対策)。
type testMsg struct {
	id        string
	sender    string
	text      string
	createdAt string
}
type testConv struct {
	uuid      string
	name      string
	createdAt string
	// updatedAt は会話の updated_at。空なら createdAt と同じにする。
	updatedAt string
	msgs      []testMsg
}

// conversationsJSON は Claude.ai の書き出しと同じ形の JSON 配列を作る。
func conversationsJSON(t *testing.T, convs []testConv) []byte {
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
		updatedAt := cv.updatedAt
		if updatedAt == "" {
			updatedAt = cv.createdAt
		}
		arr = append(arr, map[string]any{
			"uuid":          cv.uuid,
			"name":          cv.name,
			"created_at":    cv.createdAt,
			"updated_at":    updatedAt,
			"chat_messages": msgs,
		})
	}
	data, err := json.MarshalIndent(arr, "", " ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

// testZipName は実物の Claude.ai の ZIP 名。中身は conversations.json 1本。
const testZipName = "conversations-000.zip"

// testZipModified は ZIP エントリの更新時刻。実物は 1980-01-01（ZIP の元期）で固定なので、
// 「更新時刻では中身の変化を判定できない」という前提もそのまま再現される。
var testZipModified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// writeZip は dir/zipName に entries を詰めた ZIP を書く（既にあれば作り直す）。
// バイナリの fixture は commit せず、テストのたびに作る。
func writeZip(t *testing.T, dir, zipName string, entries map[string][]byte, modified time.Time) string {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for _, name := range slices.Sorted(maps.Keys(entries)) {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.Modified = modified
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := entry.Write(entries[name]); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	zipPath := filepath.Join(dir, zipName)
	if err := os.WriteFile(zipPath, buffer.Bytes(), 0o600); err != nil {
		t.Fatalf("write %s: %v", zipPath, err)
	}
	return zipPath
}

// writeConversationsAt は実物と同じ形（conversations-000.zip の中に conversations.json）の ZIP を dir へ書く。
func writeConversationsAt(t *testing.T, dir string, convs []testConv, modified time.Time) string {
	t.Helper()
	return writeZip(t, dir, testZipName, map[string][]byte{
		"conversations.json": conversationsJSON(t, convs),
	}, modified)
}

// writeConversations は既定の更新時刻で writeConversationsAt を呼ぶ。
func writeConversations(t *testing.T, dir string, convs []testConv) string {
	t.Helper()
	return writeConversationsAt(t, dir, convs, testZipModified)
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

// messageIDs はキャッシュにあるメッセージ ID の集合を返す。
func messageIDs(t *testing.T, c *pluginCache, pluginDir string) map[string]bool {
	t.Helper()
	msgs, err := c.GetMessages(pluginDir)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	ids := map[string]bool{}
	for _, m := range msgs {
		ids[m.MsgID] = true
	}
	return ids
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
	patterns := []string{dir}

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

	if err := c.build(pluginDir, patterns); err == nil {
		t.Fatal("expected build error from injected failure")
	}

	seen := messageIDs(t, c, pluginDir)
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
	patterns := []string{dir}

	writeConversations(t, dir, []testConv{
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
	})
	if err := c.build(pluginDir, patterns); err != nil {
		t.Fatalf("build 1: %v", err)
	}
	if got := countMessages(t, c); got != 1 {
		t.Fatalf("after build 1 want 1 msg, got %d", got)
	}

	// 署名が変わらなければ作り直さない(no-op)。
	if err := c.build(pluginDir, patterns); err != nil {
		t.Fatalf("build 2 (no-op): %v", err)
	}
	if got := countMessages(t, c); got != 1 {
		t.Fatalf("after no-op build want 1 msg, got %d", got)
	}

	// 会話を追加(エントリの CRC32 とサイズが変わり署名も変わる)→ 作り直しで2件に。
	writeConversations(t, dir, []testConv{
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
		{uuid: "c2", name: "two", createdAt: "2021-01-02T00:00:00Z", msgs: []testMsg{{id: "m2", sender: "human", text: "beta gamma delta", createdAt: "2021-01-02T00:00:00Z"}}},
	})
	if err := c.build(pluginDir, patterns); err != nil {
		t.Fatalf("build 3: %v", err)
	}
	if got := countMessages(t, c); got != 2 {
		t.Fatalf("after adding a conv want 2 msgs, got %d", got)
	}

	// 会話を削除(c1 を消す)→ gen 掃除で m1 が落ち、1件に。
	writeConversations(t, dir, []testConv{
		{uuid: "c2", name: "two", createdAt: "2021-01-02T00:00:00Z", msgs: []testMsg{{id: "m2", sender: "human", text: "beta gamma delta", createdAt: "2021-01-02T00:00:00Z"}}},
	})
	if err := c.build(pluginDir, patterns); err != nil {
		t.Fatalf("build 4: %v", err)
	}
	if got := countMessages(t, c); got != 1 {
		t.Fatalf("after removing a conv want 1 msg, got %d", got)
	}
	if messageIDs(t, c, pluginDir)["m1"] {
		t.Error("removed conversation's message (m1) should have been cleaned up by gen sweep")
	}
}

// 中身が同じなら ZIP を作り直しても（エントリの更新時刻が変わっても）作り直さない。
// 署名は Path:CRC32:Size で、更新時刻を見ない（実物の ZIP は 1980 年固定で当てにならない）。
func TestClaudeAISameContentDifferentMtimeDoesNotRebuild(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	patterns := []string{dir}
	convs := []testConv{
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
	}

	writeConversations(t, dir, convs)
	if err := c.build(pluginDir, patterns); err != nil {
		t.Fatalf("build 1: %v", err)
	}
	genBefore := c.getMeta("build_gen")

	// 同じ内容を別の更新時刻で作り直す（ZIP のバイト列は変わる）。
	writeConversationsAt(t, dir, convs, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	ingestConvHook = func(convID string) error {
		t.Errorf("unchanged source must not be re-ingested (conv %s)", convID)
		return nil
	}
	defer func() { ingestConvHook = nil }()
	if err := c.build(pluginDir, patterns); err != nil {
		t.Fatalf("build 2: %v", err)
	}
	if got := c.getMeta("build_gen"); got != genBefore {
		t.Errorf("build_gen moved %s -> %s although only the entry mtime changed", genBefore, got)
	}
}

// 展開済みの JSON は読まない。ZIP を消して conversations.json を直置きすると
// 「ZIP が見つかりません」で既存キャッシュは残り、設定画面向けに問題が記録される。
// conversations.json をファイルとして直接指定しても同じ（旧配置の指定を黙って0件にしない）。
func TestClaudeAILooseJSONIsNotRead(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	zipPath := writeConversations(t, dir, []testConv{
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
	})
	if err := c.build(pluginDir, []string{dir}); err != nil {
		t.Fatalf("build 1: %v", err)
	}
	if err := os.Remove(zipPath); err != nil {
		t.Fatalf("remove zip: %v", err)
	}
	loosePath := filepath.Join(dir, "conversations.json")
	if err := os.WriteFile(loosePath, conversationsJSON(t, []testConv{
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
		{uuid: "c2", name: "two", createdAt: "2021-01-02T00:00:00Z", msgs: []testMsg{{id: "m2", sender: "human", text: "beta", createdAt: "2021-01-02T00:00:00Z"}}},
	}), 0o600); err != nil {
		t.Fatalf("write loose json: %v", err)
	}

	for _, tc := range []struct {
		name     string
		patterns []string
		wantPath string
	}{
		{name: "folder", patterns: []string{dir}, wantPath: dir},
		{name: "file", patterns: []string{loosePath}, wantPath: loosePath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := c.build(pluginDir, tc.patterns)
			if err == nil || !strings.Contains(err.Error(), "ZIP") {
				t.Fatalf("want a ZIP-not-found error, got %v", err)
			}
			if got := countMessages(t, c); got != 1 {
				t.Errorf("existing cache must survive: want 1 msg, got %d", got)
			}
			problems := c.loadSourceProblems()
			if len(problems) != 1 {
				t.Fatalf("want 1 source problem, got %+v", problems)
			}
			if problems[0].Kind != string(sdk.ProblemExtractedFolder) {
				t.Errorf("kind = %s, want %s", problems[0].Kind, sdk.ProblemExtractedFolder)
			}
			if filepath.Clean(problems[0].Path) != filepath.Clean(tc.wantPath) {
				t.Errorf("path = %s, want %s", problems[0].Path, tc.wantPath)
			}
			if !strings.Contains(problems[0].Message, "解凍せず") {
				t.Errorf("message should tell the user not to extract: %s", problems[0].Message)
			}
		})
	}
}

// 会話ファイルを含まない ZIP（projects-000.zip / users-000.zip など）が並んでいても読み飛ばす。問題としても出さない。
func TestClaudeAIUnrelatedZipIsIgnored(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	writeConversations(t, dir, []testConv{
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
	})
	writeZip(t, dir, "projects-000.zip", map[string][]byte{"projects.json": []byte("[]")}, testZipModified)
	writeZip(t, dir, "users-000.zip", map[string][]byte{"users.json": []byte("[]")}, testZipModified)

	if err := c.build(pluginDir, []string{dir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := countMessages(t, c); got != 1 {
		t.Errorf("want 1 msg, got %d", got)
	}
	if got := c.getMeta("file_count"); got != "1" {
		t.Errorf("file_count = %s, want 1 (only the conversation entry counts)", got)
	}
	if problems := c.loadSourceProblems(); len(problems) != 0 {
		t.Errorf("unrelated zips must not be reported: %+v", problems)
	}
}

// 分割された ZIP（conversations-000.zip + conversations-001.zip）は全部読んで和集合にする。
func TestClaudeAISplitZipsAreMerged(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	writeZip(t, dir, "conversations-000.zip", map[string][]byte{
		"conversations.json": conversationsJSON(t, []testConv{
			{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
		}),
	}, testZipModified)
	writeZip(t, dir, "conversations-001.zip", map[string][]byte{
		"conversations.json": conversationsJSON(t, []testConv{
			{uuid: "c2", name: "two", createdAt: "2021-01-02T00:00:00Z", msgs: []testMsg{{id: "m2", sender: "human", text: "beta", createdAt: "2021-01-02T00:00:00Z"}}},
		}),
	}, testZipModified)

	if err := c.build(pluginDir, []string{dir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	ids := messageIDs(t, c, pluginDir)
	if !ids["m1"] || !ids["m2"] {
		t.Errorf("both zips should be read, got %v", ids)
	}
	if got := c.getMeta("file_count"); got != "2" {
		t.Errorf("file_count = %s, want 2", got)
	}
}

// 同じ会話が2つの ZIP にあれば updated_at が新しい版を採る（ZIP 名の順ではない）。
// 採らなかった版のメッセージは持ち越さず、片方にしか無い会話は残る。
func TestClaudeAINewestUpdatedAtWinsAcrossZips(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	// a.zip のほうが Path 順で先だが、c1 はこちらが新しい。
	writeZip(t, dir, "a.zip", map[string][]byte{
		"conversations.json": conversationsJSON(t, []testConv{
			{uuid: "c1", name: "newer", createdAt: "2021-01-01T00:00:00Z", updatedAt: "2021-03-01T00:00:00Z",
				msgs: []testMsg{{id: "m1-new", sender: "human", text: "new", createdAt: "2021-02-01T00:00:00Z"}}},
			{uuid: "c2", name: "only-a", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m2", sender: "human", text: "a", createdAt: "2021-01-01T00:00:00Z"}}},
		}),
	}, testZipModified)
	writeZip(t, dir, "b.zip", map[string][]byte{
		"conversations.json": conversationsJSON(t, []testConv{
			{uuid: "c1", name: "older", createdAt: "2021-01-01T00:00:00Z", updatedAt: "2021-01-01T00:00:00Z",
				msgs: []testMsg{{id: "m1-old", sender: "human", text: "old", createdAt: "2021-01-01T00:00:00Z"}}},
			{uuid: "c3", name: "only-b", createdAt: "2021-04-01T00:00:00Z", msgs: []testMsg{{id: "m3", sender: "human", text: "b", createdAt: "2021-04-01T00:00:00Z"}}},
		}),
	}, testZipModified)

	if err := c.build(pluginDir, []string{dir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	ids := messageIDs(t, c, pluginDir)
	for _, want := range []string{"m1-new", "m2", "m3"} {
		if !ids[want] {
			t.Errorf("%s should be present", want)
		}
	}
	if ids["m1-old"] {
		t.Error("the older version's message must not be carried over")
	}
	title, _, err := c.GetMsgByID(pluginDir, "m1-new")
	if err != nil {
		t.Fatalf("GetMsgByID: %v", err)
	}
	if title != "newer" {
		t.Errorf("conversation title = %q, want the newer version's", title)
	}
}

// source_dirs が空のときの既定（プラグインフォルダ自身）には manifest.json / config.json があるが、
// それを「展開済みフォルダ」として警告しない。ZIP を置けばそこから読める。
func TestClaudeAIPluginDirDefaultDoesNotWarn(t *testing.T) {
	c, pluginDir := newTestCache(t)
	for name, body := range map[string]string{"manifest.json": "{}", "config.json": `{"source_dirs":[]}`} {
		if err := os.WriteFile(filepath.Join(pluginDir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	patterns := parseSourcePatterns(nil, pluginDir)

	if err := c.build(pluginDir, patterns); err == nil {
		t.Fatal("no zip yet: want an error")
	}
	if problems := c.loadSourceProblems(); len(problems) != 0 {
		t.Errorf("the plugin folder itself must not be reported as an extracted folder: %+v", problems)
	}

	writeConversations(t, pluginDir, []testConv{
		{uuid: "c1", name: "one", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m1", sender: "human", text: "alpha", createdAt: "2021-01-01T00:00:00Z"}}},
	})
	if err := c.build(pluginDir, patterns); err != nil {
		t.Fatalf("build with zip in plugin dir: %v", err)
	}
	if got := countMessages(t, c); got != 1 {
		t.Errorf("want 1 msg, got %d", got)
	}
}

// 分割形式（conversations-NNN.json）と旧形式（conversations.json）の優先は**アーカイブ単位**。
// 同じ ZIP に両方あれば旧形式は読まず、別の ZIP の旧形式はそのまま読む（chatgpt と同じ規則。
// 実装は同じなのに claudeai 側にだけテストが無く、片方だけ壊れても気付けなかった）。
func TestClaudeAINumberedPreferredWithinArchive(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	writeZip(t, dir, "a-both.zip", map[string][]byte{
		"conversations.json": conversationsJSON(t, []testConv{
			{uuid: "c-old", name: "old", createdAt: "2021-01-01T00:00:00Z", msgs: []testMsg{{id: "m-old", sender: "human", text: "old", createdAt: "2021-01-01T00:00:00Z"}}},
		}),
		"conversations-000.json": conversationsJSON(t, []testConv{
			{uuid: "c-new", name: "new", createdAt: "2021-02-01T00:00:00Z", msgs: []testMsg{{id: "m-new", sender: "human", text: "new", createdAt: "2021-02-01T00:00:00Z"}}},
		}),
		"conversations-001.json": conversationsJSON(t, []testConv{
			{uuid: "c-new2", name: "new2", createdAt: "2021-03-01T00:00:00Z", msgs: []testMsg{{id: "m-new2", sender: "human", text: "new2", createdAt: "2021-03-01T00:00:00Z"}}},
		}),
	}, testZipModified)
	writeZip(t, dir, "b-legacy.zip", map[string][]byte{
		"conversations.json": conversationsJSON(t, []testConv{
			{uuid: "c-legacy", name: "legacy", createdAt: "2021-04-01T00:00:00Z", msgs: []testMsg{{id: "m-legacy", sender: "human", text: "legacy", createdAt: "2021-04-01T00:00:00Z"}}},
		}),
	}, testZipModified)

	if err := c.build(pluginDir, []string{dir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	ids := messageIDs(t, c, pluginDir)
	for _, want := range []string{"m-new", "m-new2", "m-legacy"} {
		if !ids[want] {
			t.Errorf("%s should be present", want)
		}
	}
	if ids["m-old"] {
		t.Error("conversations.json next to numbered files in the same zip must be skipped")
	}
	if got := c.getMeta("file_count"); got != "3" {
		t.Errorf("file_count = %s, want 3", got)
	}
}
