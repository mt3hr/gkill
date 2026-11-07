package main

import (
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/chatview"
)

// renderMessageHTML は1メッセージのHTMLを生成する。
//
// 描画は SDK の chatview に任せ、ここでは ChatGPT の書き出しの形を共通モデルへ写すだけにする（ADR-0315）。
// 1件は人間の発言1つか ChatGPT の返事1つの吹き出し。書き出しにはツール実行や思考が無いので本文だけになり、
// チップは返事を書いたモデル名だけ（会話タイトルは見出しに出す）。
func renderMessageHTML(convTitle string, msg cachedMessage) string {
	return chatview.Render(toChatView(convTitle, msg))
}

// toChatView は cachedMessage を chatview の共通モデルへ写す。
func toChatView(convTitle string, msg cachedMessage) chatview.Message {
	out := chatview.Message{
		Title:       convTitle,
		Chips:       []string{msg.Model},
		Role:        chatview.RoleAssistant,
		Sender:      "ChatGPT",
		Text:        strings.TrimSpace(msg.Text),
		RelatedTime: unixToTimeFromCache(msg.RelatedTimeUnix),
	}
	if msg.Sender == "user" {
		out.Role = chatview.RoleHuman
		out.Sender = "あなた"
	}
	return out
}
