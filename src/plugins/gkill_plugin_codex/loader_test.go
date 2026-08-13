package main

import (
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

const (
	parentThreadID = "00000000-0000-7000-8000-000000000001"
	subThreadID    = "00000000-0000-7000-8000-000000000002"
	oldThreadID    = "00000000-0000-7000-8000-000000000003"
)

func parentFixture() string {
	return filepath.Join("testdata", "sessions", "2026", "01", "02",
		"rollout-2026-01-02T10-00-00-"+parentThreadID+".jsonl")
}

func subAgentFixture() string {
	return filepath.Join("testdata", "sessions", "2026", "01", "02",
		"rollout-2026-01-02T10-05-00-"+subThreadID+".jsonl")
}

func oldFormatFixture() string {
	return filepath.Join("testdata", "sessions", "2026", "01", "01",
		"rollout-2026-01-01T09-00-00-"+oldThreadID+".jsonl")
}

func mustParse(t *testing.T, path string) parsedRollout {
	t.Helper()
	parsed, err := parseRollout(path)
	if err != nil {
		t.Fatalf("parseRollout(%s) = %v", path, err)
	}
	return parsed
}

func itemsOfKind(items []threadItem, kind string) []threadItem {
	var found []threadItem
	for _, item := range items {
		if item.Kind == kind {
			found = append(found, item)
		}
	}
	return found
}

func TestClassifyFileName(t *testing.T) {
	cases := map[string]string{
		"rollout-2026-01-02T10-00-00-" + parentThreadID + ".jsonl": kindRollout,
		"session_index.jsonl":         kindIndex,
		"transcription-history.jsonl": kindOther,
		"not_a_rollout.jsonl":         kindOther,
		".codex-global-state.json":    kindOther,
		"history.jsonl":               kindOther,
		"rollout-broken.jsonl":        kindOther,
	}
	for name, want := range cases {
		if got := classifyFileName(filepath.Join("x", "y", name)); got != want {
			t.Errorf("classifyFileName(%s) = %q, want %q", name, got, want)
		}
	}
}

func TestThreadIDFromFileName(t *testing.T) {
	// スレッドIDはファイル名のuuid。session_meta.session_id は使えない
	// (実データ52ファイル中23ファイルに存在せず、サブエージェントでは親を指す)。
	if got := threadIDFromFileName(parentFixture()); got != parentThreadID {
		t.Errorf("got %q, want %q", got, parentThreadID)
	}
	if got := threadIDFromFileName("rollout-2026-01-02T10-00-00-NOTAUUID.jsonl"); got != "" {
		t.Errorf("uuidでないものからIDを作ってはいけない: %q", got)
	}
	// 日時部分に区切りが多くても末尾のuuidだけを取る
	if got := threadIDFromFileName("rollout-2026-01-02T10-00-00-00000000-0000-7000-8000-00000000000A.jsonl"); got != "00000000-0000-7000-8000-00000000000a" {
		t.Errorf("小文字化されていない: %q", got)
	}
}

func TestParseRolloutTakesIdentityFromFirstMetaOnly(t *testing.T) {
	// これが最大の罠の回帰テスト。
	// サブエージェントのファイルには2つ目に「親の」session_meta が入っている。
	// identity をマージすると自分が親にすり替わり、親子のKyouIDが衝突する。
	parsed := mustParse(t, subAgentFixture())

	if parsed.Meta.ThreadID != subThreadID {
		t.Errorf("ThreadID = %q, want %q (1つ目の session_meta の id)", parsed.Meta.ThreadID, subThreadID)
	}
	if parsed.Meta.ParentThreadID != parentThreadID {
		t.Errorf("ParentThreadID = %q, want %q", parsed.Meta.ParentThreadID, parentThreadID)
	}
	if !parsed.Meta.IsSubAgent {
		t.Error("IsSubAgent が false")
	}
	if parsed.Meta.AgentNickname != "Singer" || parsed.Meta.AgentPath != "/root/explore" {
		t.Errorf("エージェント情報が取れていない: %+v", parsed.Meta)
	}
	// environment は逆に、1つ目が空で2つ目に入っているのでマージして拾う
	if parsed.Meta.Branch != "main" {
		t.Errorf("Branch = %q, want main (2つ目の session_meta からマージするべき)", parsed.Meta.Branch)
	}
	if parsed.Meta.RepoURL == "" {
		t.Error("RepoURL がマージされていない")
	}
}

func TestParseRolloutOldFormatWithoutSessionID(t *testing.T) {
	// 0.104 世代には session_id も thread_source も無い
	parsed := mustParse(t, oldFormatFixture())
	if parsed.Meta.ThreadID != oldThreadID {
		t.Errorf("ThreadID = %q, want %q", parsed.Meta.ThreadID, oldThreadID)
	}
	if parsed.Meta.IsSubAgent {
		t.Error("thread_source が無いものをサブエージェント扱いしてはいけない")
	}
	if parsed.Meta.ParentThreadID != "" {
		t.Errorf("ParentThreadID = %q, want empty", parsed.Meta.ParentThreadID)
	}
	if len(itemsOfKind(parsed.Items, itemUser)) != 1 {
		t.Error("古い版でも発言が読めるべき")
	}
}

func TestParseRolloutSkipsToolResults(t *testing.T) {
	// 実データではバイトの94.7%がツールの実行結果。1バイトも保持しない。
	parsed := mustParse(t, parentFixture())
	for _, item := range parsed.Items {
		for _, forbidden := range []string{
			"ツールの実行結果",
			"暗号化されていて読めない",
			"圧縮前の履歴",
			"これは人間が打った入力ではない",
		} {
			if strings.Contains(item.Text, forbidden) || strings.Contains(item.Extra, forbidden) {
				t.Errorf("捨てるべき内容が %s に残っている: %q", item.Kind, item.Text)
			}
		}
	}
	if parsed.Dropped != 0 {
		t.Errorf("Dropped = %d, want 0", parsed.Dropped)
	}
}

func TestParseRolloutUsesEventMsgLane(t *testing.T) {
	// 会話は event_msg レーンだけを使う。
	// response_item/message role=user には環境プリアンブルの注入が混ざる。
	parsed := mustParse(t, parentFixture())
	users := itemsOfKind(parsed.Items, itemUser)
	if len(users) != 2 {
		t.Fatalf("人間の発言が %d 件。event_msg/user_message の2件だけであるべき", len(users))
	}
	if !strings.Contains(users[0].Text, "バグを直してください") {
		t.Errorf("1つ目 = %q", users[0].Text)
	}
	if users[1].Text != "ありがとう。" {
		t.Errorf("2つ目 = %q", users[1].Text)
	}

	assistants := itemsOfKind(parsed.Items, itemAssistant)
	if len(assistants) != 2 {
		t.Errorf("応答本文が %d 件", len(assistants))
	}
}

func TestParseRolloutItemKinds(t *testing.T) {
	parsed := mustParse(t, parentFixture())
	want := map[string]int{
		itemUser:      2,
		itemAssistant: 2,
		itemThinking:  1,
		itemTool:      3, // shell_command / apply_patch / spawn_agent
		itemPatch:     1,
		itemPlan:      1,
		itemSpawn:     1,
		itemNotice:    1, // context_compacted
		itemTaskDone:  1,
		itemTurn:      1,
	}
	got := map[string]int{}
	for _, item := range parsed.Items {
		got[item.Kind]++
	}
	for kind, count := range want {
		if got[kind] != count {
			t.Errorf("%s = %d, want %d", kind, got[kind], count)
		}
	}
	for kind, count := range got {
		if _, ok := want[kind]; !ok {
			t.Errorf("知らない種別 %s が %d 件出ている", kind, count)
		}
	}
	if len(parsed.UnknownKinds) != 0 {
		t.Errorf("UnknownKinds = %v (compacted は既知の捨て対象)", parsed.UnknownKinds)
	}
}

func TestParseRolloutSeqIsLineIndex(t *testing.T) {
	// Seq は行番号。追記のみのログでは安定していて、ここが KyouID の土台になる。
	parsed := mustParse(t, parentFixture())
	previous := int64(-1)
	for _, item := range parsed.Items {
		if item.Seq <= previous {
			t.Fatalf("Seq が単調増加していない: %d のあとに %d", previous, item.Seq)
		}
		previous = item.Seq
	}
	users := itemsOfKind(parsed.Items, itemUser)
	if users[0].Seq != 2 {
		t.Errorf("1つ目の発言の Seq = %d, want 2 (0起点の行番号)", users[0].Seq)
	}
}

func TestParseRolloutKeepsPlanItem(t *testing.T) {
	parsed := mustParse(t, parentFixture())
	plans := itemsOfKind(parsed.Items, itemPlan)
	if len(plans) != 1 {
		t.Fatalf("計画が %d 件", len(plans))
	}
	if !strings.Contains(plans[0].Text, "## 計画") {
		t.Errorf("計画の本文 = %q", plans[0].Text)
	}
}

func TestStripIDEContext(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantBody   string
		wantActive string
		wantTabs   int
	}{
		{
			name: "Activeとタブの両方",
			input: "# Context from my IDE setup:\n\n## Active file: main.go\n\n## Open tabs:\n" +
				"- main.go: src/main.go\n- util.go: src/util.go\n\n## My request for Codex:\n直してください。",
			wantBody: "直してください。", wantActive: "main.go", wantTabs: 2,
		},
		{
			name: "タブのみ",
			input: "# Context from my IDE setup:\n\n## Open tabs:\n- a.ts: src/a.ts\n\n" +
				"## My request for Codex:\nこれは？",
			wantBody: "これは？", wantActive: "", wantTabs: 1,
		},
		{
			name:     "Activeのみ",
			input:    "# Context from my IDE setup:\n\n## Active file: Untitled-1\n\n## My request for Codex:\nやって",
			wantBody: "やって", wantActive: "Untitled-1", wantTabs: 0,
		},
		{
			name:     "前置きが無い",
			input:    "ふつうの発言です。",
			wantBody: "ふつうの発言です。", wantActive: "", wantTabs: 0,
		},
		{
			name:     "本文中にマーカー文字列があるだけ",
			input:    "## My request for Codex: と書いてあるログの話をしています",
			wantBody: "## My request for Codex: と書いてあるログの話をしています",
		},
		{
			name:     "前置きの形が知らないもの(マーカーが無い)",
			input:    "# Context from my IDE setup:\n\n## Open tabs:\n- a.ts: src/a.ts\n",
			wantBody: "# Context from my IDE setup:\n\n## Open tabs:\n- a.ts: src/a.ts\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, ideCtx := stripIDEContext(c.input)
			if body != c.wantBody {
				t.Errorf("body = %q, want %q", body, c.wantBody)
			}
			if c.wantActive == "" && c.wantTabs == 0 {
				if ideCtx != nil {
					t.Errorf("ideContext = %+v, want nil", ideCtx)
				}
				return
			}
			if ideCtx == nil {
				t.Fatal("ideContext が nil")
			}
			if ideCtx.ActiveFile != c.wantActive {
				t.Errorf("ActiveFile = %q, want %q", ideCtx.ActiveFile, c.wantActive)
			}
			if len(ideCtx.OpenTabs) != c.wantTabs {
				t.Errorf("OpenTabs = %d, want %d", len(ideCtx.OpenTabs), c.wantTabs)
			}
		})
	}
}

func TestSummarizeFunctionCall(t *testing.T) {
	cases := []struct {
		name string
		tool string
		args string
		want string
	}{
		{"配列のcommand", "shell_command", `{"command":["go","test","./..."]}`, "go test ./..."},
		{"文字列のcmd", "exec_command", `{"cmd":"grep -r foo ."}`, "grep -r foo ."},
		{"agent_path", "spawn_agent", `{"agent_path":"/root/explore","prompt":"調べて"}`, "/root/explore"},
		{"message", "send_message", `{"message":"終わりました"}`, "終わりました"},
		{"JSONでない", "wait", `not json`, "not json"},
		{"知らないキーだけ", "mystery", `{"zzz":1}`, `{"zzz":1}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := summarizeFunctionCall(c.tool, c.args); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestSummarizeApplyPatch(t *testing.T) {
	// 生のまま切ると "*** Begin Patch" しか見えないので、対象ファイルを拾う
	input := "*** Begin Patch\n*** Update File: src/a.go\n@@\n-x\n+y\n*** Add File: src/b.ts\n+z\n*** End Patch"
	got := summarizeCustomToolCall("apply_patch", input)
	if got != "2ファイル: a.go, b.ts" {
		t.Errorf("got %q", got)
	}

	// apply_patch でない custom_tool_call はそのまま1行に潰す
	got = summarizeCustomToolCall("exec", "const r = await tools.shell_command({});\ntext(r)")
	if !strings.HasPrefix(got, "const r = await") {
		t.Errorf("got %q", got)
	}
}

func TestSummarizeToolInputTruncates(t *testing.T) {
	long := strings.Repeat("あ", maxToolSummaryRunes+50)
	got := summarizeCustomToolCall("exec", long)
	runes := []rune(got)
	if len(runes) != maxToolSummaryRunes+1 {
		t.Errorf("長さ %d, want %d (…込み)", len(runes), maxToolSummaryRunes+1)
	}
	if !strings.HasSuffix(got, "…") {
		t.Error("末尾に … が無い")
	}
}

func TestPatchFilesOf(t *testing.T) {
	// Codexのログには "c:\..." と "C:\..." が混在するので大小を無視して相対化する
	parsed := mustParse(t, parentFixture())
	patches := itemsOfKind(parsed.Items, itemPatch)
	if len(patches) != 1 {
		t.Fatalf("patch が %d 件", len(patches))
	}
	files := decodePatchFiles(patches[0].Extra)
	if len(files) != 2 {
		t.Fatalf("変更ファイルが %d 件", len(files))
	}

	byPath := map[string]patchFile{}
	for _, file := range files {
		byPath[file.Path] = file
	}
	main, ok := byPath["src/main.go"]
	if !ok {
		t.Fatalf("相対化されていない: %v", byPath)
	}
	if main.Type != "update" || main.Added != 2 || main.Removed != 1 {
		t.Errorf("main.go = %+v, want update +2 -1 (--- +++ は数えない)", main)
	}
	newFile, ok := byPath["src/new.go"]
	if !ok {
		t.Fatalf("大文字のドライブレターが相対化されていない: %v", byPath)
	}
	if newFile.Type != "add" || newFile.Added != 1 {
		t.Errorf("new.go = %+v", newFile)
	}
}

func TestRelativizePath(t *testing.T) {
	cases := []struct{ absolute, cwd, want string }{
		{`c:\work\myproj\src\main.go`, `C:\work\myproj`, "src/main.go"},
		{`C:\work\myproj\src\main.go`, `c:\work\myproj`, "src/main.go"},
		{`C:\other\x.go`, `c:\work\myproj`, "C:/other/x.go"},
		{`/home/me/proj/a.go`, `/home/me/proj`, "a.go"},
		{`/home/me/proj/a.go`, ``, "/home/me/proj/a.go"},
	}
	for _, c := range cases {
		if got := relativizePath(c.absolute, c.cwd); got != c.want {
			t.Errorf("relativizePath(%q, %q) = %q, want %q", c.absolute, c.cwd, got, c.want)
		}
	}
}

func TestDiffCounts(t *testing.T) {
	added, removed := diffCounts("--- a\n+++ b\n@@\n-one\n+two\n+three\n ctx\n")
	if added != 2 || removed != 1 {
		t.Errorf("added=%d removed=%d, want 2/1 (--- と +++ は数えない)", added, removed)
	}
}

func TestLastPathElement(t *testing.T) {
	// filepath.Base を使わないのは、Windowsで書かれたログをLinuxで読むと
	// 区切りが解釈されずパス全体がプロジェクト名になるため
	cases := map[string]string{
		`c:\work\myproj`:  "myproj",
		`/home/me/proj`:   "proj",
		`c:\work\myproj\`: "myproj",
		`myproj`:          "myproj",
		``:                "",
	}
	for input, want := range cases {
		if got := lastPathElement(input); got != want {
			t.Errorf("lastPathElement(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestScanSources(t *testing.T) {
	source := sdk.ExpandSourcePatterns([]string{"testdata"})
	files, err := scanSources(source)
	if err != nil {
		t.Fatalf("scanSources = %v", err)
	}

	byKind := map[string]int{}
	for _, file := range files {
		byKind[file.Kind]++
	}
	if byKind[kindRollout] != 3 {
		t.Errorf("rollout = %d, want 3", byKind[kindRollout])
	}
	if byKind[kindIndex] != 1 {
		t.Errorf("index = %d, want 1", byKind[kindIndex])
	}
	if byKind[kindOther] != 0 {
		t.Errorf("対象外のファイルを拾っている: %d", byKind[kindOther])
	}

	// 新しい順(パス降順)に並ぶ。直近のデータが数秒で見えるようにするため
	for i := 1; i < len(files); i++ {
		if files[i-1].Path < files[i].Path {
			t.Errorf("並び順が降順でない: %s のあとに %s", files[i-1].Path, files[i].Path)
			break
		}
	}
}

func TestReadSessionIndex(t *testing.T) {
	titles, err := readSessionIndex(filepath.Join("testdata", "session_index.jsonl"))
	if err != nil {
		t.Fatalf("readSessionIndex = %v", err)
	}
	if titles[parentThreadID] != "バグ修正スレッド" {
		t.Errorf("親のスレッド名 = %q", titles[parentThreadID])
	}
	if titles[oldThreadID] != "古い版のスレッド" {
		t.Errorf("古い版のスレッド名 = %q", titles[oldThreadID])
	}
	// 実データでも52セッション中33件しか載っていない。存在しないIDが混ざっても落ちないこと
	if len(titles) != 3 {
		t.Errorf("件数 = %d, want 3", len(titles))
	}
	if titles[subThreadID] != "" {
		t.Error("載っていないスレッドに名前が付いている")
	}
}
