package main

import (
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/chatview"
)

// renderMessageHTML は1メッセージのHTMLを生成する。
//
// 描画は SDK の chatview に任せ、ここでは Claude.ai の書き出しの形を共通モデルへ写すだけにする（ADR-0315）。
// 1件は人間の発言1つか Claude の返事1つの吹き出し。書き出しにはツール実行・思考・モデル名が無いので
// 本文だけになり、チップは出ない（会話タイトルは見出しに出す）。
func renderMessageHTML(convTitle string, msg cachedMessage) string {
	return chatview.Render(toChatView(convTitle, msg))
}

// toChatView は cachedMessage を chatview の共通モデルへ写す。
func toChatView(convTitle string, msg cachedMessage) chatview.Message {
	out := chatview.Message{
		Title:       convTitle,
		Role:        chatview.RoleAssistant,
		Sender:      "Claude",
		Text:        strings.TrimSpace(msg.Text),
		RelatedTime: unixToTimeFromCache(msg.RelatedTimeUnix),
	}
	if msg.Sender == "human" {
		out.Role = chatview.RoleHuman
		out.Sender = "あなた"
	}
	return out
}
