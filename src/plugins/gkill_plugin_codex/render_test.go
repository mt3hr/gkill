package main

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMessageHTMLEscapes(t *testing.T) {
	// ツールの要約には生のシェルとJS、変更ファイルのパスには記号が普通に入る。
	// 補間する文字列は全部エスケープすること。
	built := message{
		Role:    roleHuman,
		Title:   `<b>タイトル</b>`,
		Project: `<proj>`,
		Text:    `<script>alert("xss")</script>`,
		IDEContext: &ideContext{
			ActiveFile: `<active>.go`,
			OpenTabs:   []ideTab{{Name: `<tab>`, Path: `src/<x>.go`}},
		},
		Items: []turnItem{
			{Kind: blockTools, Tools: []toolCall{{Name: `<tool>`, Summary: `echo "<x>" && rm -rf /`}}},
			{Kind: blockPatch, Files: []patchFile{{Path: `src/<a>.go`, Type: "update", Added: 1, Removed: 2}}},
			{Kind: blockPlan, Text: `<plan>`},
			{Kind: blockNotice, Text: `<notice>`},
		},
	}
	html := renderMessageHTML(built)

	for _, forbidden := range []string{
		`<script>alert`, `<b>タイトル</b>`, `<proj>`, `<tool>`, `<active>.go`,
		`src/<a>.go`, `<plan>`, `<notice>`, `"<x>"`,
	} {
		if strings.Contains(html, forbidden) {
			t.Errorf("エスケープされていない: %q", forbidden)
		}
	}
	for _, want := range []string{
		`&lt;script&gt;alert`, `&lt;b&gt;タイトル`, `&lt;tool&gt;`, `src/&lt;a&gt;.go`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("エスケープ後の文字列が無い: %q", want)
		}
	}
}

func TestRenderHumanHidesIDEContextBehindDetails(t *testing.T) {
	// rykv は一覧の行にこのHTMLをそのまま描く。
	// 実データでは178件中108件に前置きが付いているので、
	// 本文の前に出すとどの行も「開いているタブ一覧」で埋まって読めなくなる。
	built := message{
		Role: roleHuman,
		Text: "本文はここです",
		IDEContext: &ideContext{
			ActiveFile: "main.go",
			OpenTabs:   []ideTab{{Name: "main.go", Path: "src/main.go"}, {Name: "util.go", Path: "src/util.go"}},
		},
		RelatedTime: time.Date(2026, 1, 2, 1, 0, 0, 0, time.UTC),
	}
	html := renderMessageHTML(built)

	bodyAt := strings.Index(html, "本文はここです")
	ideAt := strings.Index(html, "IDEのコンテキスト")
	if bodyAt < 0 || ideAt < 0 {
		t.Fatalf("本文=%d IDE=%d", bodyAt, ideAt)
	}
	if ideAt < bodyAt {
		t.Error("IDEの前置きが本文より前に出ている")
	}
	if !strings.Contains(html, `<details><summary>`) {
		t.Error("IDEの前置きが折りたたまれていない")
	}
	if !strings.Contains(html, "タブ 2件") {
		t.Error("タブ数がラベルに出ていない")
	}
	if !strings.Contains(html, "あなた") {
		t.Error("人間の発言のラベルが無い")
	}
	if !strings.Contains(html, "src/util.go") {
		t.Error("タブのパスが出ていない")
	}
}

func TestRenderAssistantFoldsToolsAndSubAgent(t *testing.T) {
	built := message{
		Role:       roleAssistant,
		Project:    "myproj",
		Branch:     "main",
		Model:      "gpt-5.3-codex",
		DurationMs: 192000,
		Items: []turnItem{
			{Kind: blockThinking, Thinking: []string{"考え1", "考え2"}},
			{Kind: blockText, Text: "説明します"},
			{Kind: blockTools, Tools: []toolCall{
				{Name: "exec", Summary: "go test"},
				{Name: "exec", Summary: "go build"},
				{Name: "spawn_agent", Summary: "/root/explore", CallID: "call_1", Agent: &subAgent{
					Nickname: "Singer", AgentPath: "/root/explore", Prompt: "調べて",
					Items: []turnItem{
						{Kind: blockText, Text: "調べました"},
						{Kind: blockTools, Tools: []toolCall{{Name: "exec", Summary: "grep"}}},
					},
				}},
			}},
			{Kind: blockPatch, Files: []patchFile{
				{Path: "src/a.go", Type: "update", Added: 3, Removed: 1},
				{Path: "src/b.go", Type: "add", Added: 10},
			}},
			{Kind: blockPlan, Text: "## 計画"},
		},
	}
	html := renderMessageHTML(built)

	for _, want := range []string{
		"Codex",                  // 応答のラベル
		`class="chip">myproj`,    // チップ
		"gpt-5.3-codex",          // モデル
		"所要 3分12秒",               // 所要時間
		"exec ×2",                // ツール集計
		"💭 thinking (2)",         // 思考
		"🔧 ",                     // ツール
		"🤖 Singer /root/explore", // サブエージェント
		"調べて",                    // サブエージェントへの指示
		"📝 変更したファイル (2)",         // 変更ファイル
		"src/a.go",
		`class="patch-add">+3`,
		`class="patch-del">-1`,
		"📋 計画",
		"変更 2ファイル", // 要約行
	} {
		if !strings.Contains(html, want) {
			t.Errorf("%q が出ていない", want)
		}
	}

	// テーマ追従とiframe自動リサイズの仕掛けが入っていること
	for _, want := range []string{"gkill_iframe_size", "gkill_theme", "data-theme", "ResizeObserver"} {
		if !strings.Contains(html, want) {
			t.Errorf("%q が無い", want)
		}
	}
}

func TestRenderRespectsSizeCap(t *testing.T) {
	built := message{Role: roleAssistant}
	for range 20000 {
		built.Items = append(built.Items, turnItem{Kind: blockText, Text: strings.Repeat("あ", 500)})
	}
	html := renderMessageHTML(built)
	if len(html) > maxHTMLBytes+1024*1024 {
		t.Errorf("HTMLが %d バイト。上限 %d を大きく超えている", len(html), maxHTMLBytes)
	}
	if !strings.Contains(html, "以降は長いため省略しました") {
		t.Error("打ち切りの断りが無い")
	}
}

func TestRenderNotFoundHTML(t *testing.T) {
	html := renderNotFoundHTML()
	if !strings.Contains(html, "見つかりません") {
		t.Error("本文が無い")
	}
	if !strings.HasSuffix(html, "</body></html>") {
		t.Error("HTMLが閉じていない")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[int64]string{
		0:      "",
		999:    "",
		1000:   "所要 1秒",
		59000:  "所要 59秒",
		60000:  "所要 1分0秒",
		192000: "所要 3分12秒",
	}
	for ms, want := range cases {
		if got := formatDuration(ms); got != want {
			t.Errorf("formatDuration(%d) = %q, want %q", ms, got, want)
		}
	}
}

func TestRenderSummaryLineEmptyWhenNothingToCount(t *testing.T) {
	if got := renderSummaryLine([]turnItem{{Kind: blockText, Text: "x"}}); got != "" {
		t.Errorf("本文だけの応答に要約行を出している: %q", got)
	}
}
