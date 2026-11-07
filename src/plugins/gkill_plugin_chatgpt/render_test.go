package main

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMessageHTML(t *testing.T) {
	at := time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local)

	// 人間の発言: 会話タイトルが見出し、ラベルは「あなた」、インラインコードは <code>、モデルのチップは無い
	human := renderMessageHTML("会話の題", cachedMessage{MsgID: "m1", Sender: "user", Text: " こんにちは `go test` ", RelatedTimeUnix: at.Unix()})
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
	if strings.Contains(human, `class="chips"`) {
		t.Error("モデル名の無い発言にチップの枠を出している")
	}

	// 返事: ラベルは「ChatGPT」、モデル名がチップ、本文はエスケープ
	assistant := renderMessageHTML("", cachedMessage{MsgID: "m2", Sender: "assistant", Text: `<script>alert(1)</script>`, Model: "test-model-slug"})
	for _, want := range []string{
		`<div class="msg assistant"><div class="sender">ChatGPT</div>&lt;script&gt;alert(1)&lt;/script&gt;`,
		`<div class="chips"><span class="chip">test-model-slug</span></div>`,
	} {
		if !strings.Contains(assistant, want) {
			t.Errorf("%q が出ていない:\n%s", want, assistant)
		}
	}
	if strings.Contains(assistant, "<script>alert") {
		t.Error("本文がエスケープされていない")
	}
	if strings.Contains(assistant, `class="conv-title"`) {
		t.Error("会話タイトルが無いのに見出しを出している")
	}
}
