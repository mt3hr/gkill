package main

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMessageHTML(t *testing.T) {
	at := time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local)

	// 人間の発言: 会話タイトルが見出し、ラベルは「あなた」、インラインコードは <code>
	human := renderMessageHTML("会話の題", cachedMessage{MsgID: "m1", Sender: "human", Text: " こんにちは `go test` ", RelatedTimeUnix: at.Unix()})
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

	// 返事: ラベルは「Claude」、本文はエスケープ。書き出しにモデル名は無いのでチップも無い
	assistant := renderMessageHTML("", cachedMessage{MsgID: "m2", Sender: "assistant", Text: `<script>alert(1)</script>`})
	if !strings.Contains(assistant, `<div class="msg assistant"><div class="sender">Claude</div>&lt;script&gt;alert(1)&lt;/script&gt;`) {
		t.Errorf("返事の吹き出しの形が違う:\n%s", assistant)
	}
	if strings.Contains(assistant, "<script>alert") {
		t.Error("本文がエスケープされていない")
	}
	if strings.Contains(assistant, `class="chips"`) || strings.Contains(assistant, `class="conv-title"`) {
		t.Error("無い情報（チップ・会話タイトル）の枠を出している")
	}
}
