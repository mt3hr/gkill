package main

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMessageHTML(t *testing.T) {
	at := time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local)

	// 人間の発言: 会話タイトルが見出し、ラベルは「あなた」、インラインコードは <code>
	human := renderMessageHTML("会話の題", cachedMessage{
		MsgID: "m1", Sender: senderUser, Text: " こんにちは `go test` ", RelatedTimeUnix: at.Unix(),
	})
	for _, want := range []string{
		`<div class="conv-title">会話の題</div>`,
		`<div class="msg human"><div class="sender">あなた</div>こんにちは <code>go test</code>`,
		`<div class="ts">2024-01-01 10:00</div>`,
		"gkill_iframe_size", "gkill_theme", "data-theme", "ResizeObserver",
	} {
		if !strings.Contains(human, want) {
			t.Errorf("%q が出ていない:\n%s", want, human)
		}
	}
	// 書き出しにモデル名が入らないので、チップは常に出さない。
	if strings.Contains(human, `class="chips"`) {
		t.Error("チップの枠を出している（書き出しにモデル名は無い）")
	}

	// 返答: ラベルは「Gemini」、本文はエスケープされる
	assistant := renderMessageHTML("", cachedMessage{
		MsgID: "m2", Sender: senderModel, Text: `<script>alert(1)</script>`,
	})
	if !strings.Contains(assistant, `<div class="msg assistant"><div class="sender">Gemini</div>&lt;script&gt;alert(1)&lt;/script&gt;`) {
		t.Errorf("返答の吹き出しが出ていない:\n%s", assistant)
	}
	if strings.Contains(assistant, "<script>alert") {
		t.Error("本文がエスケープされていない")
	}
	if strings.Contains(assistant, `class="conv-title"`) {
		t.Error("会話タイトルが無いのに見出しを出している")
	}
}

// 添付（音声・ZIP・画像）は中身を読まないので、名前と注記を折りたたみに並べるだけ。
func TestRenderMessageHTMLAttachments(t *testing.T) {
	withAttachments := renderMessageHTML("", cachedMessage{
		MsgID: "m3", Sender: senderUser, Text: "これ何",
		Attachments: "00000000-0000-0000-0000-000000000001.wav\n音声が含まれています。",
	})
	for _, want := range []string{"📎 添付・注記 (2)", "00000000-0000-0000-0000-000000000001.wav", "音声が含まれています。"} {
		if !strings.Contains(withAttachments, want) {
			t.Errorf("%q が出ていない:\n%s", want, withAttachments)
		}
	}

	without := renderMessageHTML("", cachedMessage{MsgID: "m4", Sender: senderUser, Text: "ふつうの発言"})
	if strings.Contains(without, "📎") {
		t.Error("添付が無いのに折りたたみを出している")
	}
}
