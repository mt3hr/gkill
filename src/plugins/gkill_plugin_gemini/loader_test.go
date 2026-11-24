package main

import (
	"strings"
	"testing"
)

// title の接頭辞はロケールで変わる。コロン区切りと英語の "Prompted " を拾い、
// どちらでもないもの（操作の記録）はプロンプトにしない。
func TestPromptOf(t *testing.T) {
	for _, c := range []struct {
		name  string
		title string
		want  string
	}{
		{"日本語の接頭辞", "送信したメッセージ: これは何ですか", "これは何ですか"},
		{"英語のコロン形式", "Prompted: what is this", "what is this"},
		{"英語のコロン無し", "Prompted what is this", "what is this"},
		{"大小は無視する", "PROMPTED what is this", "what is this"},
		{"操作の記録はプロンプトではない", "アシスタント機能を使用", ""},
		{"英語の操作の記録", "Used Gemini Apps", ""},
		{"空", "", ""},
		{"接頭辞が長すぎるコロンは本文の一部", strings.Repeat("あ", 30) + ": つづき", ""},
		{"本文にコロンがあっても接頭辞だけ落とす", "送信したメッセージ: 比率は 1: 2 です", "比率は 1: 2 です"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := promptOf(c.title); got != c.want {
				t.Errorf("promptOf(%q) = %q, want %q", c.title, got, c.want)
			}
		})
	}
}

// 会話 ID は details の会話 URL からしか取れない（書き出しに titleUrl は無い）。
func TestConversationIDFromURL(t *testing.T) {
	for _, c := range []struct {
		url  string
		want string
	}{
		{"https://gemini.google.com/app/" + testConvA, testConvA},
		{"https://gemini.google.com/app/c/" + testConvA, testConvA},
		{"https://gemini.google.com/app/" + testConvA + "?hl=ja", testConvA},
		{"https://gemini.google.com/app/" + testConvA + "/x", testConvA},
		{"https://gemini.google.com/notebooks/view", ""},
		{"https://example.com/app/abc", ""},
		{"", ""},
	} {
		if got := conversationIDFromURL(c.url); got != c.want {
			t.Errorf("conversationIDFromURL(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

// 製品の判定は "Gemini" の部分一致。header / products / activityControls のどれか、
// それも無ければ会話 URL で見分ける。
func TestIsGeminiRecord(t *testing.T) {
	for _, c := range []struct {
		name   string
		record geminiActivity
		want   bool
	}{
		{"日本語の header", geminiActivity{Header: "Gemini アプリ"}, true},
		{"英語の header", geminiActivity{Header: "Gemini Apps"}, true},
		{"products だけ", geminiActivity{Products: []string{"Gemini アプリ"}}, true},
		{"activityControls だけ", geminiActivity{ActivityControls: []string{"Gemini アプリ アクティビティ"}}, true},
		{"会話 URL だけ", geminiActivity{Details: []geminiLink{{URL: "https://gemini.google.com/app/" + testConvA}}}, true},
		{"別の製品", geminiActivity{Header: "Chrome", Products: []string{"Chrome"}}, false},
		{"空", geminiActivity{}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := isGeminiRecord(&c.record); got != c.want {
				t.Errorf("isGeminiRecord(%+v) = %v, want %v", c.record, got, c.want)
			}
		})
	}
}

// 1レコードを1往復に直す。時刻が読めないもの・中身が無いものは落とす。
func TestExchangeOf(t *testing.T) {
	t.Run("プロンプトと返答と添付を取り出す", func(t *testing.T) {
		record := geminiActivity{
			Header: "Gemini アプリ",
			Title:  jaPromptPrefix + "これは何ですか",
			Time:   testTimeA,
			Details: []geminiLink{{
				Name: "https://gemini.google.com/app/" + testConvA,
				URL:  "https://gemini.google.com/app/" + testConvA,
			}},
			SafeHTMLItem:  []geminiHTMLItem{{HTML: "<p>こたえ</p>"}},
			AttachedFiles: []string{"00000000-0000-0000-0000-000000000001.wav"},
			Subtitles:     []geminiLink{{Name: "音声が含まれています。"}},
		}
		got, ok := exchangeOf(&record)
		if !ok {
			t.Fatal("取り出せていない")
		}
		if got.ConvID != testConvA {
			t.Errorf("ConvID = %q, want %q", got.ConvID, testConvA)
		}
		if got.Prompt != "これは何ですか" {
			t.Errorf("Prompt = %q", got.Prompt)
		}
		if got.Reply != "こたえ" {
			t.Errorf("Reply = %q", got.Reply)
		}
		if len(got.Attachments) != 2 {
			t.Errorf("Attachments = %v, want 添付名と注記の2件", got.Attachments)
		}
		// 時刻は UTC の RFC 3339 ナノ秒で持つ（Kyou ID の材料なので形を固定する）。
		if !strings.HasPrefix(got.TimeKey, "2024-01-01T00:00:00") {
			t.Errorf("TimeKey = %q", got.TimeKey)
		}
	})

	t.Run("時刻が読めない記録は落とす", func(t *testing.T) {
		record := geminiActivity{Header: "Gemini アプリ", Title: jaPromptPrefix + "x", Time: "2024年1月1日"}
		if _, ok := exchangeOf(&record); ok {
			t.Error("時刻が読めないのに取り出している")
		}
	})

	t.Run("プロンプトも返答も無い記録は落とす", func(t *testing.T) {
		record := geminiActivity{Header: "Gemini アプリ", Title: "アシスタント機能を使用", Time: testTimeA}
		if _, ok := exchangeOf(&record); ok {
			t.Error("中身が無いのに取り出している")
		}
	})
}

// 配列でない JSON・途中で切れた JSON は読まない（壊れたファイルで黙って0件にしない）。
func TestDecodeActivityArrayRejectsNonArrayAndTruncatedJSON(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
	}{
		{"オブジェクト", `{"header":"Gemini アプリ"}`},
		{"途中で切れている", `[{"header":"Gemini アプリ","time":"` + testTimeA + `"`},
		{"JSON ですらない", `not json`},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, _, err := decodeActivityArray(strings.NewReader(c.body)); err == nil {
				t.Error("エラーを返していない")
			}
		})
	}

	t.Run("先頭が Gemini でなければ読まない", func(t *testing.T) {
		body := `[{"header":"Chrome","title":"閲覧しました: x","time":"` + testTimeA + `"}]`
		records, isGemini, err := decodeActivityArray(strings.NewReader(body))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if isGemini || len(records) != 0 {
			t.Errorf("isGemini = %v, records = %d件, want false / 0件", isGemini, len(records))
		}
	})
}
