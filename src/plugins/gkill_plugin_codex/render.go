package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/chatview"
)

// renderMessageHTML は1Kyouの詳細HTMLを組み立てる。
//
// 描画は SDK の chatview に任せ、ここでは Codex のログの形を共通モデルへ写すだけにする（ADR-0315）。
// 1件は人間の発言1つか Codex の返事1つの吹き出し。返事は最終回答と質問を本文にし、
// 途中報告・思考・ツール・変更ファイルなどは「作業の経過」1つに畳む（ADR-0314）。
// rykv は一覧の行にこのHTMLをそのまま描くので、経過を開いたまま並べると会話が読めなくなる。
func renderMessageHTML(m message) string {
	return chatview.Render(toChatView(m))
}

// toChatView は message を chatview の共通モデルへ写す。
func toChatView(m message) chatview.Message {
	out := chatview.Message{
		Title:       m.Title,
		Chips:       []string{m.Project, m.Branch, m.Model, chatview.FormatDuration(time.Duration(m.DurationMs) * time.Millisecond)},
		Role:        chatview.RoleAssistant,
		Sender:      "Codex",
		Text:        m.Text,
		RelatedTime: m.RelatedTime,
	}
	if m.Role == roleHuman {
		out.Role = chatview.RoleHuman
		out.Sender = "あなた"
		if m.QuestionReply {
			out.Sender = "あなた（質問への回答）"
		}
	}

	// 前置きは本文の「後ろ」に畳んで置く。
	// rykv は一覧の行にこのHTMLをそのまま描くので、
	// 前に出すとどの行も「開いているタブ一覧」で埋まって読めなくなる。
	if fold, ok := ideContextFold(m.IDEContext); ok {
		out.Context = append(out.Context, fold)
	}
	if fold, ok := browserContextFold(m.IDEContext); ok {
		out.Context = append(out.Context, fold)
	}

	body, process := splitRunBlocks(m.Items)
	out.Body = toBlocks(body)
	out.Process = toBlocks(process)
	return out
}

// toBlocks は返事のブロック列を chatview のブロックへ写す。サブエージェントの中も同じ写し方で畳む。
func toBlocks(items []turnItem) []chatview.Block {
	blocks := make([]chatview.Block, 0, len(items))
	for _, item := range items {
		switch item.Kind {
		case blockText:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindText, Text: item.Text})
		case blockQuestion:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindQuestion, Text: item.Text})
		case blockNotice:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindNotice, Text: item.Text})
		case blockThinking:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindThinking, Thinking: item.Thinking})
		case blockPlan:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindPlan, Text: item.Text})
		case blockPatch:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindFiles, Files: toFiles(item.Files)})
		case blockTools:
			blocks = append(blocks, chatview.Block{Kind: chatview.KindTools, Tools: toToolCalls(item.Tools)})
		case blockSpawn:
			if item.Agent != nil {
				blocks = append(blocks, chatview.Block{Kind: chatview.KindAgent, Agent: toSubAgent(item.Agent)})
			}
		}
	}
	return blocks
}

func toToolCalls(tools []toolCall) []chatview.ToolCall {
	out := make([]chatview.ToolCall, 0, len(tools))
	for _, tc := range tools {
		call := chatview.ToolCall{Name: tc.Name, Summary: tc.Summary}
		if tc.Agent != nil {
			call.Agent = toSubAgent(tc.Agent)
		}
		out = append(out, call)
	}
	return out
}

func toFiles(files []patchFile) []chatview.File {
	out := make([]chatview.File, 0, len(files))
	for _, f := range files {
		out = append(out, chatview.File{Path: f.Path, Type: f.Type, Added: f.Added, Removed: f.Removed})
	}
	return out
}

// toSubAgent はサブエージェントを写す。見出しは「ニックネーム エージェントのパス」。
func toSubAgent(agent *subAgent) *chatview.SubAgent {
	label := agent.Nickname
	if label == "" {
		label = "Agent"
	}
	if agent.AgentPath != "" {
		label += " " + agent.AgentPath
	}
	return &chatview.SubAgent{Label: label, Prompt: agent.Prompt, Items: toBlocks(agent.Items)}
}

// ideContextFold はIDEの前置き（アクティブなファイルと開いているタブ）を折りたたみにする。無ければ偽。
func ideContextFold(ideCtx *ideContext) (chatview.Block, bool) {
	if ideCtx == nil || ideCtx.ActiveFile == "" && len(ideCtx.OpenTabs) == 0 {
		return chatview.Block{}, false
	}
	label := "🖥 IDEのコンテキスト"
	switch {
	case ideCtx.ActiveFile != "" && len(ideCtx.OpenTabs) != 0:
		label = fmt.Sprintf("🖥 IDEのコンテキスト (Active: %s / タブ %d件)", ideCtx.ActiveFile, len(ideCtx.OpenTabs))
	case ideCtx.ActiveFile != "":
		label = "🖥 IDEのコンテキスト (Active: " + ideCtx.ActiveFile + ")"
	case len(ideCtx.OpenTabs) != 0:
		label = fmt.Sprintf("🖥 IDEのコンテキスト (タブ %d件)", len(ideCtx.OpenTabs))
	}
	items := make([]string, 0, len(ideCtx.OpenTabs))
	for _, tab := range ideCtx.OpenTabs {
		item := tab.Name
		if tab.Path != "" {
			item += ": " + tab.Path
		}
		items = append(items, item)
	}
	return chatview.Block{Kind: chatview.KindFold, Summary: label, Items: items}, true
}

// browserContextFold はアプリ内ブラウザの前置きを折りたたみにする。無ければ偽。
func browserContextFold(preamble *ideContext) (chatview.Block, bool) {
	if preamble == nil || preamble.BrowserContext == "" {
		return chatview.Block{}, false
	}
	return chatview.Block{Kind: chatview.KindFold, Summary: "🌐 ブラウザのコンテキスト", Text: preamble.BrowserContext}, true
}

// splitRunBlocks は返事のブロックを「本文」と「作業の経過」に分ける。
//
// 本文は最終回答(phase=final_answer)と Codex からの質問。どちらも無い区切り
// (作業中に人間が割り込んだ、phase を持たない古い版)では、最後の発言を本文にする。
// 残りは時系列のまま作業の経過へ入れる。人間の発言は Items を持たないので両方空になる。
//
// Codex は質問ツールを呼ぶとき、たいてい同じ文面を最終回答にも書く。
// 最終回答が設問の書き出しを含むなら、質問は経過の側へ回して本文に二度出さない。
func splitRunBlocks(blocks []turnItem) (body []turnItem, process []turnItem) {
	var finals []string
	for _, block := range blocks {
		if block.Kind == blockText && isFinalPhase(block.Phase) {
			finals = append(finals, block.Text)
		}
	}
	restated := func(question turnItem) bool {
		lead := questionLead(question.Text)
		return lead != "" && slices.ContainsFunc(finals, func(final string) bool {
			return strings.Contains(final, lead)
		})
	}
	isBody := func(block turnItem) bool {
		if block.Kind == blockQuestion {
			return !restated(block)
		}
		return block.Kind == blockText && isFinalPhase(block.Phase)
	}
	fallback := -1
	if !slices.ContainsFunc(blocks, isBody) {
		for i := len(blocks) - 1; i >= 0; i-- {
			if blocks[i].Kind == blockText {
				fallback = i
				break
			}
		}
	}
	for i, block := range blocks {
		if isBody(block) || i == fallback {
			body = append(body, block)
		} else {
			process = append(process, block)
		}
	}
	return body, process
}

// questionLeadRunes は、最終回答が質問を言い直しているかを見るときに比べる設問の書き出しの長さ。
// 最終回答はマークダウンで書き直されることがあるので、設問の全文ではなく書き出しだけを比べる。
const questionLeadRunes = 30

// questionLead は質問ブロックの最初の設問の書き出しを返す。
func questionLead(text string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	runes := []rune(strings.TrimSpace(first))
	if len(runes) > questionLeadRunes {
		runes = runes[:questionLeadRunes]
	}
	return string(runes)
}

// formatTime は設定画面表示用に時刻を整形する。
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
