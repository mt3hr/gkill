package main

import (
	"strings"
	"testing"
)

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
	if strings.Contains(humanHTML, `class="process"`) {
		t.Error("人間の発言に作業の経過を出している")
	}

	// Claude Code の応答
	assistantHTML := renderMessageHTML(messages[2])
	for _, want := range []string{"調べます", "できました"} {
		if !strings.Contains(assistantHTML, want) {
			t.Errorf("応答本文に %q が出力されていない", want)
		}
	}
	if !strings.Contains(assistantHTML, `class="msg assistant"`) || !strings.Contains(assistantHTML, `<div class="sender">Claude Code</div>`) {
		t.Error("応答が assistant / 「Claude Code」として描画されていない")
	}
	// 最後のテキストが本文、それより前のテキスト・ツール実行は「作業の経過」の中（Codex と同じ組み方）
	processAt := strings.Index(assistantHTML, `<details class="process">`)
	if processAt < 0 {
		t.Fatal("作業の経過の折りたたみが無い")
	}
	if doneAt := strings.Index(assistantHTML, "できました"); doneAt > processAt {
		t.Error("最後のテキストが本文に出ていない（作業の経過の中にある）")
	}
	if lookAt := strings.Index(assistantHTML, "調べます"); lookAt < processAt {
		t.Error("途中のテキストが作業の経過に入っていない")
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
	// チップは project / branch / model / 所要時間。間に空白を置く(MCP の HTML→テキスト変換で "myprojmain" に連結されないため)。
	// 所要時間はフィクスチャの最初と最後の assistant レコードの差（テスト由来）
	if !strings.Contains(assistantHTML, `<span class="chip">myproj</span> <span class="chip">main</span> <span class="chip">claude-test-model</span> <span class="chip">所要 1分0秒</span>`) {
		t.Errorf("チップが project / branch / model / 所要時間の順に空白区切りで出ていない:\n%s", assistantHTML)
	}

	// HTMLエスケープの確認
	escaped := messages[1]
	escaped.Text = `<script>alert("x")</script>`
	if strings.Contains(renderMessageHTML(escaped), "<script>alert") {
		t.Error("発言本文がエスケープされていない")
	}
}

func TestSplitReplyLastTextIsBody(t *testing.T) {
	items := []turnItem{
		{Kind: "thinking", Thinking: []string{"考え"}},
		{Kind: "text", Text: "最初の報告"},
		{Kind: "tools", Tools: []toolCall{{Name: "Bash"}}},
		{Kind: "notice", Text: "通知"},
		{Kind: "text", Text: "最後の報告"},
		{Kind: "tools", Tools: []toolCall{{Name: "Edit"}}},
	}
	body, process := splitReply(items)
	if len(body) != 1 || body[0].Text != "最後の報告" {
		t.Errorf("本文は最後のテキスト1つのはず: %+v", body)
	}
	if len(process) != 5 || process[1].Text != "最初の報告" || process[4].Kind != "tools" {
		t.Errorf("残りは時系列のまま作業の経過へ入るはず: %+v", process)
	}

	// テキストが無い返事は本文が空
	body, process = splitReply([]turnItem{{Kind: "tools", Tools: []toolCall{{Name: "Bash"}}}})
	if len(body) != 0 || len(process) != 1 {
		t.Errorf("テキストが無いときは本文が空で経過だけのはず: body=%+v process=%+v", body, process)
	}
}
