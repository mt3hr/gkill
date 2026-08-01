package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	mainFixture  = "testdata/session-main.jsonl"
	agentFixture = "testdata/agent-abc123def456.jsonl"
	metaFixture  = "testdata/agent-abc123def456.meta.json"
	histFixture  = "testdata/history.jsonl"
)

// loadFixtureMessages はフィクスチャから発言を組み立てる。
func loadFixtureMessages(t *testing.T) []message {
	t.Helper()

	agentRecords, err := readRecords(agentFixture)
	if err != nil {
		t.Fatalf("error at read agent fixture: %v", err)
	}
	agentID, meta, err := readAgentMeta(metaFixture)
	if err != nil {
		t.Fatalf("error at read meta fixture: %v", err)
	}
	sa := buildSubAgent(agentID, meta, agentRecords)
	byID := map[string]*subAgent{agentID: sa}
	byToolUseID := map[string]*subAgent{meta.ToolUseID: sa}

	records, err := readRecords(mainFixture)
	if err != nil {
		t.Fatalf("error at read main fixture: %v", err)
	}
	return buildMessages(records, byToolUseID, byID)
}

func TestProbeTranscriptClassifiesFiles(t *testing.T) {
	cases := []struct {
		path        string
		wantKind    string
		wantSession string
	}{
		{mainFixture, kindMain, "S1"},
		{agentFixture, kindSubAgent, "S1"},
		{histFixture, kindOther, ""},
	}
	for _, c := range cases {
		kind, sessionID := probeTranscript(c.path)
		if kind != c.wantKind {
			t.Errorf("%s: kind = %s, want %s", c.path, kind, c.wantKind)
		}
		if sessionID != c.wantSession {
			t.Errorf("%s: sessionID = %s, want %s", c.path, sessionID, c.wantSession)
		}
	}
}

func TestBuildMessagesSplitsHumanAndResponse(t *testing.T) {
	messages := loadFixtureMessages(t)

	// 人間の発言が1件、それに続く一連の応答がまとめて1件になる
	want := []struct {
		id   string
		role string
	}{
		{"A0", roleAssistant}, // 人間の発言より前の応答
		{"U1", roleHuman},     // 最初の質問
		{"A1", roleAssistant}, // 「調べます」〜「できました」までを1件にまとめる
		{"U4", roleHuman},     // 2つめの質問
		{"A6", roleAssistant},
	}
	if len(messages) != len(want) {
		var got []string
		for _, m := range messages {
			got = append(got, m.ID+"/"+m.Role)
		}
		t.Fatalf("messages = %v (%d件), want %d件", got, len(messages), len(want))
	}
	for i, w := range want {
		if messages[i].ID != w.id || messages[i].Role != w.role {
			t.Errorf("messages[%d] = %s/%s, want %s/%s",
				i, messages[i].ID, messages[i].Role, w.id, w.role)
		}
	}

	// 人間の発言は本文が Text に入る
	if messages[1].Text != "最初の質問" {
		t.Errorf("人間の発言 = %q, want 最初の質問", messages[1].Text)
	}
	// contentがブロック配列でもテキストを取り出せる
	if messages[3].Text != "2つめの質問" {
		t.Errorf("ブロック配列の発言 = %q, want 2つめの質問", messages[3].Text)
	}
	// 応答側は Text を使わず Items に入れる
	if messages[2].Text != "" {
		t.Errorf("応答の Text = %q, want 空", messages[2].Text)
	}
	// セッション情報は全件に付く
	for _, m := range messages {
		if m.SessionTitle != "テストセッション" || m.Project != "myproj" || m.Branch != "main" {
			t.Errorf("%s: セッション情報が欠けている (%q/%q/%q)", m.ID, m.SessionTitle, m.Project, m.Branch)
		}
	}
}

func TestBuildMessagesGroupsWholeResponse(t *testing.T) {
	messages := loadFixtureMessages(t)

	// 分かれていた応答(thinking → 調べます → ツール → 通知 → できました)が1件にまとまる
	res := messages[2]
	if res.ID != "A1" {
		t.Fatalf("messages[2].ID = %s, want A1", res.ID)
	}
	var kinds []string
	for _, item := range res.Items {
		kinds = append(kinds, item.Kind)
	}
	wantKinds := []string{"thinking", "text", "tools", "notice", "text"}
	if strings.Join(kinds, ",") != strings.Join(wantKinds, ",") {
		t.Fatalf("応答の要素 = %v, want %v", kinds, wantKinds)
	}

	if res.Items[1].Text != "調べます" {
		t.Errorf("最初のテキスト = %q, want 調べます", res.Items[1].Text)
	}
	if res.Items[4].Text != "できました" {
		t.Errorf("最後のテキスト = %q, want できました", res.Items[4].Text)
	}
	if len(res.Items[0].Thinking) != 1 || res.Items[0].Thinking[0] != "考え中" {
		t.Errorf("thinking = %v, want [考え中]", res.Items[0].Thinking)
	}
	tools := res.Items[2].Tools
	if len(tools) != 2 {
		t.Fatalf("tools = %d件, want 2 (BashとAgent)", len(tools))
	}
	if tools[0].Name != "Bash" || tools[0].Summary != "ls -la" {
		t.Errorf("tools[0] = %+v, want Bash / ls -la", tools[0])
	}

	// 開始時刻は最初のレコード、更新時刻は最後に取り込んだレコード
	if got := res.RelatedTime.UTC().Format("15:04:05"); got != "01:00:10" {
		t.Errorf("RelatedTime = %s, want 01:00:10", got)
	}
	if got := res.UpdateTime.UTC().Format("15:04:05"); got != "01:01:10" {
		t.Errorf("UpdateTime = %s, want 01:01:10", got)
	}
}

func TestBuildMessagesAbsorbsSystemPrompt(t *testing.T) {
	messages := loadFixtureMessages(t)

	// task-notification は独立したKyouにならず、応答の注記として入る
	for _, m := range messages {
		if strings.Contains(m.Text, "task-notification") {
			t.Fatalf("%s: task-notification がKyouになっている", m.ID)
		}
	}
	var notices int
	for _, item := range messages[2].Items {
		if item.Kind == "notice" {
			notices++
			if !strings.Contains(item.Text, "サブエージェントが完了しました") {
				t.Errorf("notice text = %q", item.Text)
			}
		}
	}
	if notices != 1 {
		t.Errorf("notice count = %d, want 1", notices)
	}
}

func TestBuildMessagesLinksSubAgent(t *testing.T) {
	messages := loadFixtureMessages(t)

	tools := messages[2].Items[2].Tools
	agentCall := tools[1]
	if agentCall.Name != "Agent" {
		t.Fatalf("tools[1].Name = %s, want Agent", agentCall.Name)
	}
	if agentCall.Agent == nil {
		t.Fatal("Agentツールにサブエージェントが紐付いていない")
	}
	if agentCall.Agent.AgentType != "Explore" {
		t.Errorf("AgentType = %q, want Explore", agentCall.Agent.AgentType)
	}
	if agentCall.Agent.Description != "調査" {
		t.Errorf("Description = %q, want 調査", agentCall.Agent.Description)
	}
	if agentCall.Agent.Prompt != "調べてください" {
		t.Errorf("Prompt = %q, want 調べてください", agentCall.Agent.Prompt)
	}
	if len(agentCall.Agent.Items) == 0 {
		t.Error("サブエージェントの本文が空")
	}
}

func TestScanSources(t *testing.T) {
	src := expandSourcePatterns([]string{"testdata"})
	if len(src.Dirs) != 1 {
		t.Fatalf("Dirs = %v, want 1件", src.Dirs)
	}
	files, err := scanSources(src, map[string]scannedFile{})
	if err != nil {
		t.Fatalf("error at scan: %v", err)
	}
	kinds := map[string]int{}
	for _, f := range files {
		kinds[f.Kind]++
	}
	if kinds[kindMain] != 1 {
		t.Errorf("main files = %d, want 1", kinds[kindMain])
	}
	if kinds[kindSubAgent] != 1 {
		t.Errorf("subagent files = %d, want 1", kinds[kindSubAgent])
	}
	if kinds[kindMeta] != 1 {
		t.Errorf("meta files = %d, want 1", kinds[kindMeta])
	}
	// history.jsonl はトランスクリプトではないので対象外になる
	if kinds[kindOther] != 1 {
		t.Errorf("other files = %d, want 1", kinds[kindOther])
	}
}

func TestScanSourcesReusesKnownEntries(t *testing.T) {
	src := expandSourcePatterns([]string{"testdata"})
	first, err := scanSources(src, map[string]scannedFile{})
	if err != nil {
		t.Fatalf("error at scan: %v", err)
	}
	known := map[string]scannedFile{}
	for _, f := range first {
		known[f.Path] = f
	}
	second, err := scanSources(src, known)
	if err != nil {
		t.Fatalf("error at rescan: %v", err)
	}
	if len(second) != len(first) {
		t.Fatalf("rescan count = %d, want %d", len(second), len(first))
	}
	for _, f := range second {
		prev := known[f.Path]
		if f.Kind != prev.Kind || f.SessionID != prev.SessionID {
			t.Errorf("%s: 種別が引き継がれていない (%s/%s -> %s/%s)",
				filepath.Base(f.Path), prev.Kind, prev.SessionID, f.Kind, f.SessionID)
		}
	}
}

func TestSummarizeToolInput(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"Bash", `{"command":"go test ./...","description":"テスト"}`, "go test ./..."},
		{"Read", `{"file_path":"C:\\a\\b.go","offset":10}`, `C:\a\b.go`},
		{"Grep", `{"pattern":"foo.*bar"}`, "foo.*bar"},
		{"Agent", `{"description":"調査","subagent_type":"Explore"}`, "調査"},
		{"Unknown", `{"whatever":1}`, `{"whatever":1}`},
	}
	for _, c := range cases {
		got := summarizeToolInput(c.name, []byte(c.input))
		if got != c.want {
			t.Errorf("summarizeToolInput(%s) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSummarizeToolInputTruncates(t *testing.T) {
	long := strings.Repeat("あ", maxToolSummaryRunes+50)
	got := summarizeToolInput("Bash", []byte(`{"command":"`+long+`"}`))
	if len([]rune(got)) != maxToolSummaryRunes+1 {
		t.Errorf("要約の長さ = %d, want %d (末尾の…込み)", len([]rune(got)), maxToolSummaryRunes+1)
	}
}

func TestParseSourcePatterns(t *testing.T) {
	want := []string{`C:\a`, `C:\b`}
	eq := func(got []string) bool {
		if len(got) != len(want) {
			return false
		}
		for i := range want {
			if got[i] != want[i] {
				return false
			}
		}
		return true
	}

	// 文字列(改行区切り)
	if got := parseSourcePatterns("  C:\\a  \r\n\nC:\\b\n"); !eq(got) {
		t.Errorf("文字列指定 = %v, want %v", got, want)
	}
	// 配列
	if got := parseSourcePatterns([]any{`C:\a`, `C:\b`}); !eq(got) {
		t.Errorf("配列指定 = %v, want %v", got, want)
	}
	if got := parseSourcePatterns([]string{`C:\a`, `C:\b`}); !eq(got) {
		t.Errorf("[]string指定 = %v, want %v", got, want)
	}
	// 配列の要素に改行が混ざっていても分解する
	if got := parseSourcePatterns([]any{"C:\\a\nC:\\b"}); !eq(got) {
		t.Errorf("配列+改行 = %v, want %v", got, want)
	}
	// 空・nilのときは既定のフォルダにフォールバックする
	for _, v := range []any{nil, "", []any{}} {
		if def := parseSourcePatterns(v); len(def) != 1 || def[0] != defaultSourceDir() {
			t.Errorf("空設定(%v)のフォールバック = %v, want [%s]", v, def, defaultSourceDir())
		}
	}
}

func TestExpandSourcePatterns(t *testing.T) {
	// ワイルドカード無し: ディレクトリはDirs、ファイルはFiles
	src := expandSourcePatterns([]string{"testdata", mainFixture})
	if len(src.Dirs) != 1 || len(src.Files) != 1 {
		t.Errorf("Dirs=%v Files=%v, want ディレクトリ1・ファイル1", src.Dirs, src.Files)
	}

	// ファイル名のパターン
	src = expandSourcePatterns([]string{"testdata/agent-*.jsonl"})
	if len(src.Files) != 1 || len(src.Dirs) != 0 {
		t.Errorf("agent-*.jsonl = Dirs%v Files%v, want ファイル1件", src.Dirs, src.Files)
	}

	// ** で再帰的にマッチする
	src = expandSourcePatterns([]string{"**/*.meta.json"})
	if len(src.Files) == 0 {
		t.Errorf("**/*.meta.json が何にもマッチしていない")
	}

	// 重複は畳まれる
	src = expandSourcePatterns([]string{mainFixture, mainFixture})
	if len(src.Files) != 1 {
		t.Errorf("同じファイルの重複指定 = %v, want 1件", src.Files)
	}

	// マッチしないものは Missing に入る
	src = expandSourcePatterns([]string{"testdata/does-not-exist", "testdata/*.nomatch"})
	if len(src.Missing) != 2 {
		t.Errorf("Missing = %v, want 2件", src.Missing)
	}
}

func TestExpandedPatternIsScannable(t *testing.T) {
	// パターンで拾ったファイルも、ディレクトリ走査と同じように種別判定される
	src := expandSourcePatterns([]string{"testdata/*.jsonl"})
	files, err := scanSources(src, map[string]scannedFile{})
	if err != nil {
		t.Fatalf("error at scan: %v", err)
	}
	kinds := map[string]int{}
	for _, f := range files {
		kinds[f.Kind]++
	}
	if kinds[kindMain] != 1 || kinds[kindSubAgent] != 1 || kinds[kindOther] != 1 {
		t.Errorf("種別 = %v, want main1/subagent1/other1", kinds)
	}
	// *.jsonl なので meta.json は拾わない
	if kinds[kindMeta] != 0 {
		t.Errorf("meta files = %d, want 0", kinds[kindMeta])
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("ホームディレクトリが取れない環境")
	}
	if got := parseSourcePatterns("~/.claude/projects"); len(got) != 1 ||
		got[0] != filepath.Join(home, ".claude/projects") {
		t.Errorf("~展開 = %v, want %v", got, filepath.Join(home, ".claude/projects"))
	}
	// 途中の ~ は展開しない
	if got := parseSourcePatterns(`C:\a~b`); len(got) != 1 || got[0] != `C:\a~b` {
		t.Errorf("途中の~ = %v, want C:\\a~b", got)
	}
}

func TestHasGlobMeta(t *testing.T) {
	cases := map[string]bool{
		`C:\Users\user\.claude\projects`: false,
		`C:\Users\user\PC\Claude*`:       true,
		`C:\logs\**\*.jsonl`:             true,
		`C:\logs\file?.jsonl`:            true,
		`C:\logs\[ab].jsonl`:             true,
	}
	for pattern, want := range cases {
		if got := hasGlobMeta(pattern); got != want {
			t.Errorf("hasGlobMeta(%q) = %v, want %v", pattern, got, want)
		}
	}
}

func TestRenderMessageHTMLEscapesAndFolds(t *testing.T) {
	messages := loadFixtureMessages(t)

	// 人間の発言
	humanHTML := renderMessageHTML(messages[1])
	if !strings.Contains(humanHTML, "最初の質問") {
		t.Error("人間の発言本文が出力されていない")
	}
	if !strings.Contains(humanHTML, `class="msg human"`) || !strings.Contains(humanHTML, "あなた") {
		t.Error("人間の発言が human として描画されていない")
	}

	// Claude の応答
	assistantHTML := renderMessageHTML(messages[2])
	for _, want := range []string{"調べます", "できました"} {
		if !strings.Contains(assistantHTML, want) {
			t.Errorf("応答本文に %q が出力されていない", want)
		}
	}
	if !strings.Contains(assistantHTML, `class="msg assistant"`) || !strings.Contains(assistantHTML, "Claude") {
		t.Error("Claudeの発言が assistant として描画されていない")
	}
	if !strings.Contains(assistantHTML, "<details>") {
		t.Error("折りたたみが出力されていない")
	}
	if !strings.Contains(assistantHTML, "🤖") {
		t.Error("サブエージェントの折りたたみが出力されていない")
	}
	if !strings.Contains(assistantHTML, "Bash ×1") {
		t.Error("ツール集計が出力されていない")
	}
	// ツールの実行結果は保持しない
	if strings.Contains(assistantHTML, "ツールの結果は保持しない") {
		t.Error("tool_result の内容が出力されている")
	}
	// チップは残す
	if !strings.Contains(assistantHTML, `class="chip"`) {
		t.Error("プロジェクト/ブランチのチップが出ていない")
	}

	// HTMLエスケープの確認
	escaped := messages[1]
	escaped.Text = `<script>alert("x")</script>`
	if strings.Contains(renderMessageHTML(escaped), "<script>alert") {
		t.Error("発言本文がエスケープされていない")
	}
}

func TestSearchTextIncludesMessageAndTools(t *testing.T) {
	messages := loadFixtureMessages(t)
	text := searchTextOf(messages[2])

	for _, want := range []string{"調べます", "できました", "テストセッション", "myproj", "main", "Bash", "ls -la", "Explore", "調査"} {
		if !strings.Contains(text, want) {
			t.Errorf("検索テキストに %q が含まれていない", want)
		}
	}
}
