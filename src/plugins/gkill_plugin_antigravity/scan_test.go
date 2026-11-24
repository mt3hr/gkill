package main

import (
	"os"
	"path/filepath"
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 走査は transcript_full.jsonl と conversation_summaries.db だけを拾う。
// 省略版 transcript.jsonl と chunks/ の写しを拾うと、同じ会話が二重になる。
func TestScanSourcesPicksOnlyTranscriptFull(t *testing.T) {
	files, problems, err := scanSources(sdk.ExpandSourcePatterns([]string{filepath.Join("testdata", "brain")}))
	if err != nil {
		t.Fatalf("scanSources: %v", err)
	}
	if len(problems.MissingID) != 0 || problems.Shadowed != 0 {
		t.Errorf("problems = %+v, want 空", problems)
	}

	ids := map[string]string{}
	for _, file := range files {
		if file.Kind != kindTranscript {
			t.Errorf("transcript 以外を拾っている: %+v", file)
			continue
		}
		if filepath.Base(file.Path) != transcriptFileName {
			t.Errorf("読まないはずのファイルを拾っている: %s", file.Path)
		}
		ids[file.ConversationID] = file.Path
	}
	if len(ids) != 2 {
		t.Fatalf("会話数 = %d, want 2: %v", len(ids), ids)
	}
	if _, ok := ids[conversationA]; !ok {
		t.Errorf("会話A が無い: %v", ids)
	}
	if _, ok := ids[conversationB]; !ok {
		t.Errorf("会話B が無い: %v", ids)
	}
}

// 会話IDは生の置き場でも同期先でも「直近の UUID フォルダ名」から採れる。
func TestConversationIDOfBothLayouts(t *testing.T) {
	cases := map[string]string{
		filepath.Join("home", ".gemini", "antigravity", "brain", conversationA,
			".system_generated", "logs", transcriptFileName): conversationA,
		filepath.Join("home", "Kyou", "Antigravity_TestPC_20240101", conversationA,
			transcriptFileName): conversationA,
		filepath.Join("home", "Kyou", "Antigravity_TestPC_20240101", transcriptFileName): "",
		transcriptFileName: "",
	}
	for path, want := range cases {
		if got := conversationIDOf(path); got != want {
			t.Errorf("conversationIDOf(%q) = %q, want %q", path, got, want)
		}
	}
	// 大文字で書かれていても同じ会話として扱う
	upper := filepath.Join("brain", "00000000-0000-7000-8000-00000000000A", transcriptFileName)
	if got := conversationIDOf(upper); got != "00000000-0000-7000-8000-00000000000a" {
		t.Errorf("大文字の会話ID = %q", got)
	}
}

// 会話IDのフォルダに入っていない transcript は読まず、設定画面の警告へ出す。
// パスから代用IDを振ると、別の会話と混ざる。
func TestScanSourcesReportsMissingConversationID(t *testing.T) {
	files, problems, err := scanSources(sdk.ExpandSourcePatterns([]string{filepath.Join("testdata", "loose")}))
	if err != nil {
		t.Fatalf("scanSources: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("読まないはずのファイルを拾っている: %+v", files)
	}
	if len(problems.MissingID) != 1 {
		t.Fatalf("MissingID = %v, want 1件", problems.MissingID)
	}
	if filepath.Base(problems.MissingID[0]) != transcriptFileName {
		t.Errorf("MissingID の中身 = %q", problems.MissingID[0])
	}
}

// 同じ会話が2か所にあったら、一番大きいファイルだけを読む（ログは追記のみ）。
func TestScanSourcesPicksLargestPerConversation(t *testing.T) {
	files, problems, err := scanSources(sdk.ExpandSourcePatterns([]string{
		filepath.Join("testdata", "brain", conversationA),
		filepath.Join("testdata", "synced"),
	}))
	if err != nil {
		t.Fatalf("scanSources: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("ファイル数 = %d, want 1: %+v", len(files), files)
	}
	if problems.Shadowed != 1 {
		t.Errorf("Shadowed = %d, want 1", problems.Shadowed)
	}

	large, err := os.Stat(fixtureTranscript(conversationA))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if files[0].Size != large.Size() {
		t.Errorf("小さいほうを選んでいる: size = %d, want %d", files[0].Size, large.Size())
	}
}

// ファイル名だけで種別を決める。
func TestClassifyFileName(t *testing.T) {
	cases := map[string]string{
		filepath.Join("logs", transcriptFileName):                    kindTranscript,
		filepath.Join("antigravity", summariesFileName):              kindSummaries,
		filepath.Join("logs", truncatedTranscriptFileName):           kindOther,
		filepath.Join("logs", "chunks", "transcript_full", "0.json"): kindOther,
		filepath.Join("logs", "00000000.jsonl"):                      kindOther,
	}
	for path, want := range cases {
		if got := classifyFileName(path); got != want {
			t.Errorf("classifyFileName(%q) = %q, want %q", path, got, want)
		}
	}
}

// 既定の取り込み元は ~/.gemini/antigravity の下を指す。
func TestDefaultSourcePatterns(t *testing.T) {
	for _, pattern := range defaultSourcePatternsForConfig() {
		if filepath.IsAbs(pattern) {
			t.Errorf("config.json に書く既定が絶対パスになっている: %q", pattern)
		}
	}
	patterns := defaultSourcePatterns()
	if len(patterns) == 0 {
		t.Skip("ホームが分からない環境")
	}
	for _, pattern := range patterns {
		if !filepath.IsAbs(pattern) {
			t.Errorf("既定が絶対パスでない: %q", pattern)
		}
	}
}

// workspace_uris からプロジェクト名（開いているフォルダ名）だけを取り出す。
func TestProjectNameOf(t *testing.T) {
	cases := map[string]string{
		`["file:///c:/example/repo-alpha"]`:    "repo-alpha",
		`["file:///c%3A/example/repo-alpha"]`:  "repo-alpha",
		`["file:///c:/example/repo-alpha/"]`:   "repo-alpha",
		`["file:///home/testuser/repo-alpha"]`: "repo-alpha",
		`[]`:                                   "",
		``:                                     "",
		`壊れたJSON`:                              "壊れたJSON",
	}
	for input, want := range cases {
		if got := projectNameOf(input); got != want {
			t.Errorf("projectNameOf(%q) = %q, want %q", input, got, want)
		}
	}
}
