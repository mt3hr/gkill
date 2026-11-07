package chatview

import (
	"maps"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRenderHeadContract(t *testing.T) {
	// plugin-html-view.vue との契約。テーマ追従と iframe の自動リサイズの仕掛けが入っていること。
	out := Render(Message{Role: RoleAssistant, Sender: "Codex", Text: "x"})
	for _, want := range []string{
		`<!DOCTYPE html>`, `<meta charset="utf-8">`, `[data-theme="dark"]`,
		"gkill_iframe_size", "gkill_theme", "data-theme", "ResizeObserver", "'toggle'",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q が無い", want)
		}
	}
	if !strings.HasSuffix(out, "</body></html>") {
		t.Error("HTML が閉じていない")
	}

	// postMessage のキーは2つだけ。足すときはクライアントと cross-boundary-map.md §9 も変える
	keys := map[string]bool{}
	for _, key := range regexp.MustCompile(`gkill_[a-z_]+`).FindAllString(out, -1) {
		keys[key] = true
	}
	if want := map[string]bool{"gkill_iframe_size": true, "gkill_theme": true}; !maps.Equal(keys, want) {
		t.Errorf("postMessage のキーが増減している: %v", keys)
	}
	// dblclick の転送はクライアントが本文の末尾に足す
	if strings.Contains(out, "dblclick") {
		t.Error("dblclick 転送スクリプトをプラグイン側で書いている")
	}
}

func TestRenderEscapesEveryField(t *testing.T) {
	out := Render(Message{
		Title:  `<b>題</b>`,
		Chips:  []string{`<proj>`},
		Role:   RoleHuman,
		Sender: `<you>`,
		Text:   `<script>alert("xss")</script>`,
		Context: []Block{
			{Kind: KindFold, Summary: `<fold>`, Text: `<ctx>`, Items: []string{`<item>`}},
		},
		Body: []Block{
			{Kind: KindQuestion, Text: `<q>`},
		},
		Process: []Block{
			{Kind: KindTools, Tools: []ToolCall{{Name: `<tool>`, Summary: `echo "<x>" && rm -rf /`, Agent: &SubAgent{Label: `<agent>`, Prompt: `<prompt>`}}}},
			{Kind: KindFiles, Files: []File{{Path: `src/<a>.go`, Type: `<t>`, Added: 1}}},
			{Kind: KindPlan, Text: `<plan>`},
			{Kind: KindNotice, Text: `<notice>`},
			{Kind: KindThinking, Thinking: []string{`<think>`}},
		},
	})
	for _, forbidden := range []string{
		`<b>題</b>`, `<proj>`, `<you>`, `<script>alert`, `<fold>`, `<ctx>`, `<item>`, `<q>`,
		`<tool>`, `"<x>"`, `<agent>`, `<prompt>`, `src/<a>.go`, `<t>`, `<plan>`, `<notice>`, `<think>`,
	} {
		if strings.Contains(out, forbidden) {
			t.Errorf("エスケープされていない: %q", forbidden)
		}
	}
	for _, want := range []string{`&lt;script&gt;alert`, `&lt;b&gt;題`, `&lt;tool&gt;`, `src/&lt;a&gt;.go`, `❓ &lt;you&gt; からの質問`} {
		if !strings.Contains(out, want) {
			t.Errorf("エスケープ後の文字列が無い: %q", want)
		}
	}
}

func TestRenderChipsSpaceSeparatedDedupedSkipEmpty(t *testing.T) {
	// 間に空白を置く。MCP の HTML→テキスト変換は <span> を改行にしないので、詰めると "myprojmain" に連結される
	out := Render(Message{Chips: []string{"myproj", "", "main", "myproj"}})
	if !strings.Contains(out, `<span class="chip">myproj</span> <span class="chip">main</span></div>`) {
		t.Errorf("チップが空白区切り・重複除去・空スキップで出ていない:\n%s", out)
	}
	if strings.Contains(Render(Message{Chips: []string{"", ""}}), `class="chips"`) {
		t.Error("チップが1つも無いのに枠を出している")
	}
}

func TestRenderRoleClassAndSender(t *testing.T) {
	human := Render(Message{Role: RoleHuman, Sender: "あなた", Text: "こんにちは"})
	if !strings.Contains(human, `<div class="msg human"><div class="sender">あなた</div>こんにちは`) {
		t.Errorf("人間の吹き出しの形が違う:\n%s", human)
	}
	assistant := Render(Message{Role: RoleAssistant, Sender: "Claude Code", Text: "はい"})
	if !strings.Contains(assistant, `<div class="msg assistant"><div class="sender">Claude Code</div>はい`) {
		t.Errorf("応答の吹き出しの形が違う:\n%s", assistant)
	}
}

func TestRenderTimestampLocalOrEmpty(t *testing.T) {
	at := time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local)
	if out := Render(Message{RelatedTime: at}); !strings.Contains(out, `<div class="ts">2024-01-01 10:00</div>`) {
		t.Errorf("時刻が出ていない:\n%s", out)
	}
	if out := Render(Message{}); !strings.Contains(out, `<div class="ts"></div>`) {
		t.Error("ゼロ時刻のときに空の ts が出ていない")
	}
}

func TestRenderOrderTitleChipsTextContextBodyProcessTimestamp(t *testing.T) {
	out := Render(Message{
		Title:       "セッション名",
		Chips:       []string{"myproj"},
		Role:        RoleAssistant,
		Sender:      "Codex",
		Text:        "本文はここ",
		Context:     []Block{{Kind: KindFold, Summary: "🖥 前置き", Items: []string{"a.go: src/a.go"}}},
		Body:        []Block{{Kind: KindText, Text: "最終回答"}},
		Process:     []Block{{Kind: KindText, Text: "途中経過です"}, {Kind: KindTools, Tools: []ToolCall{{Name: "exec"}, {Name: "exec"}}}, {Kind: KindThinking, Thinking: []string{"考え"}}, {Kind: KindFiles, Files: []File{{Path: "a"}, {Path: "b"}}}},
		RelatedTime: time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local),
	})
	order := []string{
		"セッション名",
		`class="chip">myproj`,
		"本文はここ",
		"🖥 前置き",
		`class="answer">最終回答`,
		`<details class="process"><summary>🛠 作業の経過 — 途中報告 ×1 · exec ×2 · thinking ×1 · 変更 2ファイル</summary>`,
		`class="commentary">途中経過です`,
		`class="ts">2024-01-01 10:00`,
	}
	last := -1
	for _, want := range order {
		at := strings.Index(out, want)
		if at < 0 {
			t.Fatalf("%q が無い:\n%s", want, out)
		}
		if at < last {
			t.Errorf("%q の位置が前の要素より前にある", want)
		}
		last = at
	}
	if strings.Contains(Render(Message{Body: []Block{{Kind: KindText, Text: "x"}}}), `class="process"`) {
		t.Error("作業の経過が空なのに折りたたみを出している")
	}
}

func TestRenderProcessKinds(t *testing.T) {
	out := Render(Message{
		Role:   RoleAssistant,
		Sender: "Codex",
		Process: []Block{
			{Kind: KindThinking, Thinking: []string{"考え1", "考え2"}},
			{Kind: KindTools, Tools: []ToolCall{
				{Name: "exec", Summary: "go test"},
				{Name: "exec", Summary: "go build"},
				{Name: "", Summary: "名無し"},
				{Name: "spawn_agent", Summary: "/root/explore", Agent: &SubAgent{
					Label: "Singer /root/explore", Prompt: "調べて",
					Items: []Block{
						{Kind: KindText, Text: "調べました"},
						{Kind: KindTools, Tools: []ToolCall{{Name: "exec", Summary: "grep"}}},
						{Kind: KindFiles, Files: []File{{Path: "x"}}},
						{Kind: KindThinking, Thinking: []string{"隠す"}},
						{Kind: KindAgent, Agent: &SubAgent{Label: "孫"}},
					},
				}},
			}},
			{Kind: KindFiles, Files: []File{
				{Path: "src/a.go", Type: "update", Added: 3, Removed: 1},
				{Path: "src/b.go", Type: "add", Added: 10},
			}},
			{Kind: KindPlan, Text: "## 計画"},
			{Kind: KindNotice, Text: "中断されました"},
			{Kind: KindQuestion, Text: "どちらにしますか？"},
			{Kind: KindAgent, Agent: &SubAgent{Prompt: "名無しの子"}},
			{Kind: KindFold, Summary: "🌐 ブラウザ", Text: "URL: x"},
		},
	})
	for _, want := range []string{
		`<details><summary>💭 thinking (2)</summary><div class="thinking-body">考え1</div><div class="thinking-body">考え2</div></details>`,
		`<details><summary>🔧 exec ×2  (不明) ×1  spawn_agent ×1</summary><ul class="tool-list">`,
		`<li><span class="tool-name">exec</span> go test</li>`,
		`<details><summary>🤖 Singer /root/explore</summary><div class="agent-prompt">調べて</div><div class="agent-body">調べました</div><div class="notice">🔧 exec ×1</div><div class="notice">📝 変更 1ファイル</div><div class="notice">🤖 孫</div></details>`,
		`<details><summary>📝 変更したファイル (2)</summary><ul class="patch-list"><li>src/a.go (update) <span class="patch-add">+3</span> <span class="patch-del">-1</span></li><li>src/b.go (add) <span class="patch-add">+10</span></li></ul></details>`,
		`<details><summary>📋 計画</summary><div class="plan-body">## 計画</div></details>`,
		`<div class="notice">中断されました</div>`,
		`<div class="question"><div class="question-label">❓ Codex からの質問</div>どちらにしますか？</div>`,
		`<details><summary>🤖 Agent</summary><div class="agent-prompt">名無しの子</div></details>`,
		`<details><summary>🌐 ブラウザ</summary><div class="plan-body">URL: x</div></details>`,
		// 要約はこの階層だけを数える（サブエージェントの中の exec は入らない）。同数は名前順
		`🛠 作業の経過 — exec ×2 · (不明) ×1 · spawn_agent ×1 · thinking ×2 · 変更 2ファイル`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q が出ていない:\n%s", want, out)
		}
	}
	if strings.Contains(out, "隠す") {
		t.Error("サブエージェントの thinking を出している")
	}
}

func TestRenderCodeOnlyInProse(t *testing.T) {
	out := Render(Message{
		Role:    RoleAssistant,
		Text:    "まず `go test` を回す",
		Process: []Block{{Kind: KindTools, Tools: []ToolCall{{Name: "exec", Summary: "`go test`"}}}},
	})
	if !strings.Contains(out, "まず <code>go test</code> を回す") {
		t.Error("本文のインラインコードが <code> になっていない")
	}
	if !strings.Contains(out, "<li><span class=\"tool-name\">exec</span> `go test`</li>") {
		t.Error("ツールの要約のバッククォートは文字のまま出す")
	}
}

func TestRenderRespectsSizeCap(t *testing.T) {
	m := Message{Role: RoleAssistant}
	for range 20000 {
		m.Process = append(m.Process, Block{Kind: KindText, Text: strings.Repeat("あ", 500)})
	}
	out := Render(m)
	if len(out) > MaxHTMLBytes+1024*1024 {
		t.Errorf("HTML が %d バイト。上限 %d を大きく超えている", len(out), MaxHTMLBytes)
	}
	if !strings.Contains(out, "以降は長いため省略しました") {
		t.Error("打ち切りの断りが無い")
	}
	if !strings.HasSuffix(out, "</body></html>") {
		t.Error("打ち切っても HTML は閉じる")
	}
}

func TestRenderNotFound(t *testing.T) {
	out := RenderNotFound()
	if !strings.Contains(out, "見つかりません") {
		t.Error("本文が無い")
	}
	if !strings.Contains(out, "gkill_theme") {
		t.Error("見つからないページもテーマに追従させる")
	}
	if !strings.HasSuffix(out, "</body></html>") {
		t.Error("HTML が閉じていない")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		0:                              "",
		999 * time.Millisecond:         "",
		time.Second:                    "所要 1秒",
		59 * time.Second:               "所要 59秒",
		time.Minute:                    "所要 1分0秒",
		3*time.Minute + 12*time.Second: "所要 3分12秒",
	}
	for d, want := range cases {
		if got := FormatDuration(d); got != want {
			t.Errorf("FormatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestSummaryLine(t *testing.T) {
	if got := SummaryLine([]Block{{Kind: KindText, Text: "x"}, {Kind: KindNotice, Text: "y"}}); got != "" {
		t.Errorf("数えるものが無いのに要約を出している: %q", got)
	}
	// 回数の多い順、同数なら名前順。thinking と変更ファイルは後ろ
	got := SummaryLine([]Block{
		{Kind: KindTools, Tools: []ToolCall{{Name: "b"}, {Name: "a"}, {Name: "b"}, {Name: "c"}}},
		{Kind: KindThinking, Thinking: []string{"1", "2"}},
		{Kind: KindFiles, Files: []File{{Path: "x"}}},
	})
	if want := "b ×2 · a ×1 · c ×1 · thinking ×2 · 変更 1ファイル"; got != want {
		t.Errorf("SummaryLine = %q, want %q", got, want)
	}
}
