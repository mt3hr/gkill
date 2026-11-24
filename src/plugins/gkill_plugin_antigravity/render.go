package main

import (
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/chatview"
)

// renderMessageHTML は1Kyouの詳細HTMLを組み立てる。
//
// 描画は SDK の chatview に任せ、ここでは Antigravity のログの形を共通モデルへ写すだけにする
// （ADR-0315）。返事は最後のテキストを本文にし、それより前のテキスト・thinking・
// ツール呼び出し・システム発の注記は「作業の経過」1つに時系列で畳む
// （Codex / Claude Code と同じ組み方。Antigravity のログには最終回答の印が無いので、
// ADR-0314 の「最終回答が無い区切りでは最後の発言を本文にする」規則をそのまま使う）。
// rykv は一覧の行にこのHTMLをそのまま描くので、途中のテキストを全部並べると会話が読めなくなる。
func renderMessageHTML(t message) string {
	return chatview.Render(toChatView(t))
}

// toChatView は message を chatview の共通モデルへ写す。
func toChatView(t message) chatview.Message {
	out := chatview.Message{
		Title:       t.Title,
		Chips:       []string{t.Project, t.Model},
		Role:        chatview.RoleAssistant,
		Sender:      "Antigravity",
		Text:        t.Text,
		RelatedTime: t.RelatedTime,
	}
	if t.Role == roleHuman {
		out.Role = chatview.RoleHuman
		out.Sender = "あなた"
	} else {
		// 所要時間は応答のまとまりの最初のステップから最後のステップまで。
		if d := t.UpdateTime.Sub(t.RelatedTime); d > 0 {
			out.Chips = append(out.Chips, chatview.FormatDuration(d))
		}
		body, process := splitReply(t.Items)
		out.Body = toBlocks(body)
		out.Process = toBlocks(process)
	}

	// 前置きは本文の「後ろ」に畳んで置く（chatview が Context をそう描く）。
	// 前に出すと、一覧の行がどれも自動で付く定型文で埋まって会話が読めなくなる。
	if fold, ok := preambleFold(t.Preamble); ok {
		out.Context = append(out.Context, fold)
	}
	return out
}

// preambleFold は人間の発言から剥がした前置きを折りたたみ1つにする。無ければ偽。
func preambleFold(preamble []string) (chatview.Block, bool) {
	text := strings.TrimSpace(strings.Join(preamble, "\n\n"))
	if text == "" {
		return chatview.Block{}, false
	}
	return chatview.Block{Kind: chatview.KindFold, Summary: "⚙ 設定の変更", Text: text}, true
}

// splitReply は応答の要素を「本文」と「作業の経過」に分ける。
// 最後のテキストが本文、残り（それより前のテキスト・thinking・ツール・注記）は時系列のまま経過へ。
// テキストが1つも無い応答（ツール呼び出しだけで切れた）は本文が空になり、経過だけが出る。
func splitReply(items []turnItem) (body []turnItem, process []turnItem) {
	last := -1
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Kind == itemText {
			last = i
			break
		}
	}
	for i, item := range items {
		if i == last {
			body = append(body, item)
		} else {
			process = append(process, item)
		}
	}
	return body, process
}

// toBlocks は応答の要素列を chatview のブロックへ写す。
func toBlocks(items []turnItem) []chatview.Block {
	blocks := make([]chatview.Block, 0, len(items))
	for _, item := range items {
		switch item.Kind {
		case itemText:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindText, Text: item.Text})
		case itemNotice:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindNotice, Text: item.Text})
		case itemThinking:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindThinking, Thinking: item.Thinking})
		case itemTools:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindTools, Tools: toToolCalls(item.Tools)})
		}
	}
	return blocks
}

func toToolCalls(tools []toolCall) []chatview.ToolCall {
	out := make([]chatview.ToolCall, 0, len(tools))
	for _, tc := range tools {
		out = append(out, chatview.ToolCall{Name: tc.Name, Summary: tc.Summary})
	}
	return out
}
