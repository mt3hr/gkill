package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustParseFixture(t *testing.T, conversationID string) parsedTranscript {
	t.Helper()
	path := fixtureTranscript(conversationID)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	parsed, err := parseTranscript(scannedFile{
		Path:           path,
		MtimeUnix:      info.ModTime().Unix(),
		Size:           info.Size(),
		Kind:           kindTranscript,
		ConversationID: conversationID,
	})
	if err != nil {
		t.Fatalf("parseTranscript: %v", err)
	}
	return parsed
}

func itemKinds(items []turnItem) []string {
	kinds := make([]string, 0, len(items))
	for _, item := range items {
		kinds = append(kinds, item.Kind)
	}
	return kinds
}

// 人間の発言1つと、それに続く一連の応答がそれぞれ1件になる。
func TestBuildMessagesSplitsPerUserInput(t *testing.T) {
	parsed := mustParseFixture(t, conversationA)
	if parsed.StepCount != 11 {
		t.Errorf("ステップ数 = %d, want 11", parsed.StepCount)
	}
	if parsed.DroppedLines != 0 {
		t.Errorf("読めなかった行 = %d, want 0", parsed.DroppedLines)
	}
	if len(parsed.Messages) != 4 {
		t.Fatalf("Kyou数 = %d, want 4: %+v", len(parsed.Messages), parsed.Messages)
	}

	roles := []string{}
	for _, m := range parsed.Messages {
		roles = append(roles, m.Role)
	}
	want := []string{roleHuman, roleAssistant, roleHuman, roleAssistant}
	if strings.Join(roles, ",") != strings.Join(want, ",") {
		t.Errorf("並び = %v, want %v", roles, want)
	}
	if parsed.Messages[0].Text != "テストの記録を作ってください" {
		t.Errorf("人間の発言 = %q", parsed.Messages[0].Text)
	}
	if parsed.Messages[1].Model != "gemini-test-model" {
		t.Errorf("モデル名 = %q", parsed.Messages[1].Model)
	}
	if parsed.Messages[1].Text != "" {
		t.Errorf("応答の Text は空にしておく（本文は描画のときに Items から採る）: %q", parsed.Messages[1].Text)
	}
}

// 最初の USER_INPUT より前のステップは、最初の応答の経過の先頭へ入れる。黙って捨てない。
func TestBuildMessagesKeepsRecordsBeforeFirstUserInput(t *testing.T) {
	parsed := mustParseFixture(t, conversationA)
	first := parsed.Messages[1]
	if len(first.Items) == 0 || first.Items[0].Kind != itemNotice {
		t.Fatalf("先頭が注記でない: %v", itemKinds(first.Items))
	}
	if !strings.Contains(first.Items[0].Text, "前の会話からの引き継ぎ") {
		t.Errorf("引き継ぎが落ちている: %q", first.Items[0].Text)
	}
	if !strings.HasPrefix(first.Items[0].Text, "会話履歴の引き継ぎ: ") {
		t.Errorf("注記の見出しが無い: %q", first.Items[0].Text)
	}
}

// USER_INPUT が1件も無い会話も応答1件にする。0件にはしない。
func TestBuildMessagesWithoutUserInput(t *testing.T) {
	parsed := mustParseFixture(t, conversationB)
	if len(parsed.Messages) != 1 {
		t.Fatalf("Kyou数 = %d, want 1", len(parsed.Messages))
	}
	built := parsed.Messages[0]
	if built.Role != roleAssistant {
		t.Errorf("role = %q, want %q", built.Role, roleAssistant)
	}
	if kinds := itemKinds(built.Items); strings.Join(kinds, ",") != "notice,text,notice" {
		t.Errorf("経過 = %v, want notice,text,notice", kinds)
	}
	// error_code が数値で入っていても行ごと読めなくならない
	if !strings.Contains(built.Items[2].Text, "エラーの詳細") {
		t.Errorf("ERROR_MESSAGE の内容が落ちている: %q", built.Items[2].Text)
	}
}

// ツールの実行結果の content は保持しない。エラーだけ1行の注記にする。
func TestBuildMessagesDropsToolResultContent(t *testing.T) {
	parsed := mustParseFixture(t, conversationA)
	for _, built := range parsed.Messages {
		for _, item := range built.Items {
			if strings.Contains(item.Text, "これは保持しない") {
				t.Errorf("ツールの実行結果を保持している: %q", item.Text)
			}
		}
		if strings.Contains(searchTextOf(built), "これは保持しない") {
			t.Error("ツールの実行結果が検索テキストに入っている")
		}
	}

	second := parsed.Messages[3]
	if len(second.Items) == 0 || second.Items[0].Kind != itemNotice {
		t.Fatalf("ツールのエラーが注記になっていない: %v", itemKinds(second.Items))
	}
	if !strings.Contains(second.Items[0].Text, "ツールが失敗しました") {
		t.Errorf("ツールのエラーが落ちている: %q", second.Items[0].Text)
	}
}

// 同じ本文の注記は1件に畳む（Antigravity は同じ定型文を1会話で何度も書く）。
func TestBuildMessagesFoldsDuplicatedNotices(t *testing.T) {
	parsed := mustParseFixture(t, conversationA)
	count := 0
	for _, item := range parsed.Messages[1].Items {
		if item.Kind == itemNotice && strings.Contains(item.Text, "毎回同じ定型の案内") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("同じ注記が %d 件並んでいる, want 1件", count)
	}
}

// thinking とツール呼び出しは経過へ入り、ツール呼び出しには要約が付く。
func TestBuildMessagesCarriesThinkingAndTools(t *testing.T) {
	parsed := mustParseFixture(t, conversationA)
	kinds := strings.Join(itemKinds(parsed.Messages[1].Items), ",")
	if kinds != "notice,thinking,tools,text,notice,text,tools" {
		t.Fatalf("経過 = %s", kinds)
	}

	tools := parsed.Messages[1].Items[2].Tools
	if len(tools) != 1 || tools[0].Name != "read_file" {
		t.Fatalf("ツール = %+v", tools)
	}
	if tools[0].Summary != "/example/sample.txt" {
		t.Errorf("要約 = %q, want パス", tools[0].Summary)
	}
	// MCP の呼び出しは サーバ名/ツール名
	mcp := parsed.Messages[1].Items[6].Tools
	if len(mcp) != 1 || mcp[0].Summary != "ExampleServer/example_tool" {
		t.Errorf("MCP の要約 = %+v", mcp)
	}
}

// Kyou ID は会話が伸びても動かない（動くと利用者が付けたタグやテキストが迷子になる）。
func TestKyouIDIsStableAcrossAppend(t *testing.T) {
	parsed := mustParseFixture(t, conversationA)

	dir := t.TempDir()
	data, err := os.ReadFile(fixtureTranscript(conversationA))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	appended := append(data,
		[]byte(`{"step_index":11,"type":"USER_INPUT","source":"USER_EXPLICIT","created_at":"2024-01-01T00:11:00Z","content":"続きの質問"}`+"\n")...)
	target := filepath.Join(dir, conversationA)
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(target, transcriptFileName)
	if err := os.WriteFile(path, appended, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	grown, err := parseTranscript(scannedFile{Path: path, Kind: kindTranscript, ConversationID: conversationA})
	if err != nil {
		t.Fatalf("parseTranscript: %v", err)
	}
	if len(grown.Messages) != len(parsed.Messages)+1 {
		t.Fatalf("追記後の Kyou数 = %d, want %d", len(grown.Messages), len(parsed.Messages)+1)
	}
	for i, before := range parsed.Messages {
		if grown.Messages[i].ID != before.ID {
			t.Errorf("%d件目の ID が動いた: %s -> %s", i, before.ID, grown.Messages[i].ID)
		}
	}

	// 発言者を種に混ぜているので、人間と応答の ID は衝突しない
	if kyouIDOf(conversationA, roleHuman, 1) == kyouIDOf(conversationA, roleAssistant, 1) {
		t.Error("人間と応答の ID が衝突している")
	}
	// ファイルの置き場所に依らない
	if kyouIDOf(conversationA, roleHuman, 1) != parsed.Messages[0].ID {
		t.Error("ID が置き場所で変わっている")
	}
}

// 検索テキストは本文・経過・thinking・ツール名・モデル名を含む。
// 会話タイトルとプロジェクト名は焼き込まない（別カラムで更新するため）。
func TestSearchTextOfCoversItems(t *testing.T) {
	parsed := mustParseFixture(t, conversationA)
	built := parsed.Messages[1]
	built.Title = "焼き込まないタイトル"
	built.Project = "repo-alpha"

	text := searchTextOf(built)
	for _, want := range []string{"どこを直すか考える", "途中の報告です", "できました", "read_file", "gemini-test-model"} {
		if !strings.Contains(text, want) {
			t.Errorf("検索テキストに %q が無い", want)
		}
	}
	for _, unwanted := range []string{"焼き込まないタイトル", "repo-alpha"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("検索テキストに %q が焼き込まれている", unwanted)
		}
	}
}

// 読めない行はファイルごと捨てずに数える。
func TestParseTranscriptCountsUnreadableLines(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, conversationA)
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(target, transcriptFileName)
	content := "これはJSONではない\n" +
		`{"step_index":0,"type":"USER_INPUT","source":"USER_EXPLICIT","created_at":"2024-01-01T00:00:00Z","content":"読める行"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	parsed, err := parseTranscript(scannedFile{Path: path, Kind: kindTranscript, ConversationID: conversationA})
	if err != nil {
		t.Fatalf("parseTranscript: %v", err)
	}
	if parsed.DroppedLines != 1 {
		t.Errorf("読めなかった行 = %d, want 1", parsed.DroppedLines)
	}
	if len(parsed.Messages) != 1 {
		t.Errorf("読める行が取り込まれていない: %d件", len(parsed.Messages))
	}
}

// 要約はファイルの中身が入る欄を使わない（長いので）。
func TestSummarizeToolArgsAvoidsFileContent(t *testing.T) {
	summary := summarizeToolArgs([]byte(`{"Path":"/example/sample.txt","Content":"` + strings.Repeat("あ", 500) + `"}`))
	if summary != "/example/sample.txt" {
		t.Errorf("要約 = %q", summary)
	}
	// toolSummary があれば最優先
	if got := summarizeToolArgs([]byte(`{"Path":"/example/a.txt","toolSummary":"例の要約"}`)); got != "例の要約" {
		t.Errorf("toolSummary を使っていない: %q", got)
	}
	// 知らない形でも1行に収める
	long := summarizeToolArgs([]byte(`{"unknown":` + strings.Repeat(`"あ",`, 1) + `"x"}`))
	if long == "" {
		t.Error("知らない形で要約が空になっている")
	}
	if got := summarizeToolArgs(nil); got != "" {
		t.Errorf("args が無いのに要約が出ている: %q", got)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("あいうえお", 3); got != "あいう…" {
		t.Errorf("truncateRunes = %q", got)
	}
	if got := truncateRunes("あいう", 3); got != "あいう" {
		t.Errorf("ちょうどのとき印が付いている: %q", got)
	}
}
