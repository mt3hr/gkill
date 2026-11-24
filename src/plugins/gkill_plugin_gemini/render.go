package main

import (
	"fmt"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/chatview"
)

// renderMessageHTML は1発言のHTMLを生成する。
//
// 描画は SDK の chatview に任せ、ここではマイアクティビティの形を共通モデルへ写すだけにする（ADR-0315）。
// 1件は人間のプロンプト1つか Gemini の返答1つの吹き出し。書き出しにはツール実行や思考が無いので
// 本文だけになり、チップも無い（書き出しにモデル名が入らない）。
// 添付ファイルがあるときだけ、名前を折りたたみに並べる。
func renderMessageHTML(convTitle string, msg cachedMessage) string {
	return chatview.Render(toChatView(convTitle, msg))
}

// toChatView は cachedMessage を chatview の共通モデルへ写す。
func toChatView(convTitle string, msg cachedMessage) chatview.Message {
	out := chatview.Message{
		Title:       convTitle,
		Role:        chatview.RoleAssistant,
		Sender:      "Gemini",
		Text:        strings.TrimSpace(msg.Text),
		RelatedTime: unixToTimeFromCache(msg.RelatedTimeUnix),
	}
	if msg.Sender == senderUser {
		out.Role = chatview.RoleHuman
		out.Sender = "あなた"
	}
	if block, ok := attachmentsBlock(msg.Attachments); ok {
		out.Context = []chatview.Block{block}
	}
	return out
}

// attachmentsBlock は添付ファイル名と注記の折りたたみを作る。
//
// プラグインは添付の中身を読まない（音声・画像・ZIP が Takeout の ZIP の中に並ぶ）。
// 何が付いていたかだけ分かればよいので、名前を並べる。
func attachmentsBlock(attachments string) (chatview.Block, bool) {
	items := make([]string, 0, 4)
	for line := range strings.SplitSeq(attachments, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			items = append(items, line)
		}
	}
	if len(items) == 0 {
		return chatview.Block{}, false
	}
	return chatview.Block{
		Kind:    chatview.KindFold,
		Summary: fmt.Sprintf("📎 添付・注記 (%d)", len(items)),
		Items:   items,
	}, true
}
