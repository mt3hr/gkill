package main

import (
	"strings"
	"testing"
	"time"
)

// 人間の発言と応答でラベルが変わり、チップは project / model / 所要時間の順に空白1つで並ぶ。
// 詰めて並べると MCP の HTML→テキスト変換でプロジェクト名とモデル名が1語に連結される。
func TestRenderMessageHTMLLabelsAndChips(t *testing.T) {
	related := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	human := renderMessageHTML(message{
		Role: roleHuman, Title: "会話のタイトル", Project: "repo-alpha",
		Text: "人間の発言", RelatedTime: related, UpdateTime: related,
	})
	if !strings.Contains(human, "あなた") {
		t.Error("人間側のラベルが無い")
	}
	if strings.Contains(human, "Antigravity") {
		t.Error("人間の発言に応答側のラベルが出ている")
	}
	if !strings.Contains(human, "会話のタイトル") {
		t.Error("会話のタイトルが出ていない")
	}

	reply := renderMessageHTML(message{
		Role: roleAssistant, Title: "会話のタイトル", Project: "repo-alpha",
		Model: "gemini-test-model", RelatedTime: related, UpdateTime: related.Add(90 * time.Second),
		Items: []turnItem{{Kind: itemText, Text: "返事"}},
	})
	if !strings.Contains(reply, "Antigravity") {
		t.Error("応答側のラベルが無い")
	}
	project := strings.Index(reply, "repo-alpha")
	model := strings.Index(reply, "gemini-test-model")
	if project < 0 || model < 0 || project > model {
		t.Errorf("チップの順が project, model になっていない: project=%d model=%d", project, model)
	}
	if !strings.Contains(reply, "repo-alpha</span> <span") {
		t.Errorf("チップが空白1つで区切られていない: %s", reply)
	}
}

// 返事は最後のテキストが本文で、それより前は「作業の経過」へ畳む。
func TestSplitReplyTakesLastText(t *testing.T) {
	items := []turnItem{
		{Kind: itemNotice, Text: "注記"},
		{Kind: itemText, Text: "途中の報告"},
		{Kind: itemTools, Tools: []toolCall{{Name: "read_file", Summary: "/example/sample.txt"}}},
		{Kind: itemText, Text: "最終の返事"},
	}
	body, process := splitReply(items)
	if len(body) != 1 || body[0].Text != "最終の返事" {
		t.Errorf("本文 = %+v", body)
	}
	if len(process) != 3 {
		t.Errorf("経過 = %d件, want 3件", len(process))
	}

	// テキストが1つも無ければ本文は空で、経過だけが出る
	body, process = splitReply([]turnItem{{Kind: itemTools, Tools: []toolCall{{Name: "read_file"}}}})
	if len(body) != 0 || len(process) != 1 {
		t.Errorf("テキストの無い応答: body=%d process=%d", len(body), len(process))
	}
}

// 本文は必ずエスケープする。経過は折りたたみの中に入る。
func TestRenderMessageHTMLEscapesAndFolds(t *testing.T) {
	html := renderMessageHTML(message{
		Role: roleAssistant,
		Items: []turnItem{
			{Kind: itemText, Text: "途中の報告"},
			{Kind: itemText, Text: "<script>alert(1)</script>"},
		},
	})
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Error("エスケープされていない")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("エスケープした文字列が無い: %s", html)
	}
	if !strings.Contains(html, "作業の経過") {
		t.Error("経過の折りたたみが無い")
	}
}

// 人間の吹き出しには依頼だけが出る。剥がした前置きは本文の後ろの折りたたみへ。
func TestRenderMessageHTMLHumanShowsRequestOnly(t *testing.T) {
	related := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	html := renderMessageHTML(message{
		Role:        roleHuman,
		Title:       "会話のタイトル",
		Text:        "依頼の本文",
		Preamble:    []string{"自動で付く定型文。設定が変わったことの通知"},
		RelatedTime: related,
		UpdateTime:  related,
	})

	if !strings.Contains(html, "依頼の本文") {
		t.Error("依頼が出ていない")
	}
	for _, unwanted := range []string{tagUserRequest, tagAdditionalMetadata, "current local time"} {
		if strings.Contains(html, unwanted) {
			t.Errorf("包みの跡が出ている: %q", unwanted)
		}
	}
	if !strings.Contains(html, "設定の変更") {
		t.Errorf("前置きの折りたたみの見出しが無い: %s", html)
	}
	if !strings.Contains(html, "自動で付く定型文") {
		t.Error("前置きの中身が落ちている")
	}
	// 折りたたみ（details）の中に入っていること。本文より後ろに出る
	if strings.Index(html, "依頼の本文") > strings.Index(html, "自動で付く定型文") {
		t.Error("前置きが本文より前に出ている（一覧の行が定型文で埋まる）")
	}
	if !strings.Contains(html, "<details") {
		t.Error("前置きが折りたたみになっていない")
	}

	// 前置きが無ければ折りたたみを出さない
	plain := renderMessageHTML(message{Role: roleHuman, Text: "依頼だけ", RelatedTime: related})
	if strings.Contains(plain, "設定の変更") {
		t.Error("前置きが無いのに折りたたみが出ている")
	}
}
