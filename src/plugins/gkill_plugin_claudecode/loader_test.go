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

// loadFixtureTurns はフィクスチャからターンを組み立てる。
func loadFixtureTurns(t *testing.T) []turn {
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
	return buildTurns(records, byToolUseID, byID)
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

func TestBuildTurnsSplitsOnHumanPrompts(t *testing.T) {
	turns := loadFixtureTurns(t)

	if len(turns) != 2 {
		t.Fatalf("turns = %d, want 2", len(turns))
	}
	if turns[0].ID != "U1" {
		t.Errorf("turns[0].ID = %s, want U1", turns[0].ID)
	}
	if turns[1].ID != "U4" {
		t.Errorf("turns[1].ID = %s, want U4", turns[1].ID)
	}
	if turns[0].Prompt != "最初の質問" {
		t.Errorf("turns[0].Prompt = %q, want 最初の質問", turns[0].Prompt)
	}
	// contentがブロック配列の人間プロンプトもテキストを取り出せること
	if turns[1].Prompt != "2つめの質問" {
		t.Errorf("turns[1].Prompt = %q, want 2つめの質問", turns[1].Prompt)
	}
	if turns[0].SessionTitle != "テストセッション" {
		t.Errorf("SessionTitle = %q, want テストセッション", turns[0].SessionTitle)
	}
	if turns[0].Project != "myproj" {
		t.Errorf("Project = %q, want myproj", turns[0].Project)
	}
	if turns[0].Branch != "main" {
		t.Errorf("Branch = %q, want main", turns[0].Branch)
	}
	// 最初の人間プロンプトより前のassistantレコードは捨てられること
	for _, item := range turns[0].Items {
		if strings.Contains(item.Text, "捨てられる") {
			t.Errorf("最初の人間プロンプトより前の要素がターンに含まれている")
		}
	}
	// UpdateTimeはターン最終レコードの時刻
	if got := turns[0].UpdateTime.UTC().Format("15:04:05"); got != "01:01:10" {
		t.Errorf("turns[0].UpdateTime = %s, want 01:01:10", got)
	}
}

func TestBuildTurnsAbsorbsSystemPrompt(t *testing.T) {
	turns := loadFixtureTurns(t)

	// task-notification は独立したターンにならず、直前のターンにnoticeとして入る
	var notices int
	for _, item := range turns[0].Items {
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

func TestBuildTurnsGroupsToolsAndThinking(t *testing.T) {
	turns := loadFixtureTurns(t)

	var kinds []string
	for _, item := range turns[0].Items {
		kinds = append(kinds, item.Kind)
	}
	want := []string{"thinking", "text", "tools", "notice", "text"}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("item kinds = %v, want %v", kinds, want)
	}

	tools := turns[0].Items[2].Tools
	if len(tools) != 2 {
		t.Fatalf("tools = %d, want 2 (BashとAgentが連続しているのでまとまる)", len(tools))
	}
	if tools[0].Name != "Bash" || tools[0].Summary != "ls -la" {
		t.Errorf("tools[0] = %+v, want Bash / ls -la", tools[0])
	}
}

func TestBuildTurnsLinksSubAgent(t *testing.T) {
	turns := loadFixtureTurns(t)

	tools := turns[0].Items[2].Tools
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
		`C:\Users\user\DevPC\Claude*`:   true,
		`C:\logs\**\*.jsonl`:              true,
		`C:\logs\file?.jsonl`:             true,
		`C:\logs\[ab].jsonl`:              true,
	}
	for pattern, want := range cases {
		if got := hasGlobMeta(pattern); got != want {
			t.Errorf("hasGlobMeta(%q) = %v, want %v", pattern, got, want)
		}
	}
}

func TestRenderTurnHTMLEscapesAndFolds(t *testing.T) {
	turns := loadFixtureTurns(t)
	html := renderTurnHTML(turns[0])

	if !strings.Contains(html, "最初の質問") {
		t.Error("プロンプトが出力されていない")
	}
	if !strings.Contains(html, "<details>") {
		t.Error("折りたたみが出力されていない")
	}
	if !strings.Contains(html, "🤖") {
		t.Error("サブエージェントの折りたたみが出力されていない")
	}
	if !strings.Contains(html, "Bash ×1") {
		t.Error("ツール集計が出力されていない")
	}
	// ツールの実行結果は保持しない
	if strings.Contains(html, "ツールの結果は保持しない") {
		t.Error("tool_result の内容が出力されている")
	}

	// HTMLエスケープの確認
	escaped := turns[0]
	escaped.Prompt = `<script>alert("x")</script>`
	if strings.Contains(renderTurnHTML(escaped), "<script>alert") {
		t.Error("プロンプトがエスケープされていない")
	}
}

func TestSearchTextIncludesPromptAndTools(t *testing.T) {
	turns := loadFixtureTurns(t)
	text := searchTextOf(turns[0])

	for _, want := range []string{"最初の質問", "テストセッション", "調べます", "Bash", "ls -la", "Explore", "調査"} {
		if !strings.Contains(text, want) {
			t.Errorf("検索テキストに %q が含まれていない", want)
		}
	}
}
