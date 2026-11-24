package main

import (
	"strings"
	"testing"
)

// 返答は HTML で届くが、chatview は本文を RenderText（コードフェンスとインラインコードだけ）に
// 通すので、markdown 風の平文へ写してから渡す。
// 実物の書き出しに出てくるタグ（p / strong / li / code / ul / ol / h2-h4 / table /
// pre / a / em / br / hr / blockquote）を網羅する。
func TestHTMLToText(t *testing.T) {
	for _, c := range []struct {
		name string
		html string
		want string
	}{
		{"段落", "<p>ひとつ</p><p>ふたつ</p>", "ひとつ\n\nふたつ"},
		{"強調", "<p>これは<strong>太字</strong>です</p>", "これは**太字**です"},
		{"斜体", "<p><em>斜体</em></p>", "*斜体*"},
		{"インラインコード", "<p><code>go test</code> を実行</p>", "`go test` を実行"},
		{"見出し", "<h3>みだし</h3><p>本文</p>", "### みだし\n\n本文"},
		{"箇条書き", "<ul><li>あ</li><li>い</li></ul>", "- あ\n- い"},
		{"番号つき", "<ol><li>あ</li><li>い</li></ol>", "1. あ\n2. い"},
		{"入れ子の箇条書き", "<ul><li>親<ul><li>子</li></ul></li></ul>", "- 親\n  - 子"},
		{"改行", "<p>うえ<br>した</p>", "うえ\nした"},
		{"水平線", "<p>a</p><hr><p>b</p>", "a\n\n---\n\nb"},
		{"引用", "<blockquote>引用文</blockquote>", "> 引用文"},
		{"コードブロック", "<pre><code>line1\nline2</code></pre>", "```\nline1\nline2\n```"},
		{"表", "<table><tr><th>A</th><th>B</th></tr><tr><td>1</td><td>2</td></tr></table>", "| A | B |\n| 1 | 2 |"},
		{"空", "", ""},
		{"タグだけ", "<p></p>", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := htmlToText(c.html); got != c.want {
				t.Errorf("htmlToText(%q)\n got = %q\nwant = %q", c.html, got, c.want)
			}
		})
	}
}

// 実体参照は戻す。タグは残さない（chatview が全部エスケープするので、残すと文字として出る）。
func TestHTMLToTextUnescapesAndDropsTags(t *testing.T) {
	got := htmlToText("<p>a &amp; b &lt;c&gt; &quot;d&quot; &#39;e&#39;&nbsp;f</p>")
	for _, want := range []string{"&", "<c>", `"d"`, "'e'"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q が戻っていない: %q", want, got)
		}
	}
	if strings.Contains(got, "&amp;") || strings.Contains(got, "<p>") {
		t.Errorf("タグか実体参照が残っている: %q", got)
	}
}

// リンクは本文のあとに URL を括弧で添える。本文がそのまま URL なら添えない。
func TestHTMLToTextLinks(t *testing.T) {
	got := htmlToText(`<p><a href="https://example.com/x">リンク</a></p>`)
	if got != "リンク (https://example.com/x)" {
		t.Errorf("got %q", got)
	}
	same := htmlToText(`<p><a href="https://example.com/x">https://example.com/x</a></p>`)
	if same != "https://example.com/x" {
		t.Errorf("URL が二重になっている: %q", same)
	}
}

// 知らないタグは中身のテキストだけ拾う。壊れた断片でも落ちない。
func TestHTMLToTextUnknownTagsAndBrokenInput(t *testing.T) {
	if got := htmlToText("<span>なかみ</span>"); got != "なかみ" {
		t.Errorf("知らないタグ: %q", got)
	}
	if got := htmlToText("<p>切れて"); got != "切れて" {
		t.Errorf("閉じていない: %q", got)
	}
	if got := htmlToText("<<<>>>"); strings.Contains(got, "panic") {
		t.Errorf("壊れた入力: %q", got)
	}
}

// 単語の間の空白は落とさない（タグをまたぐと詰まりやすい）。
func TestHTMLToTextKeepsSpacesAcrossTags(t *testing.T) {
	if got := htmlToText("<p>a <strong>b</strong> c</p>"); got != "a **b** c" {
		t.Errorf("got %q, want %q", got, "a **b** c")
	}
}

// 極端に長い入力でも上限で止まる（壊れた入力で無限に伸びないため）。
func TestHTMLToTextStopsAtLimit(t *testing.T) {
	long := "<p>" + strings.Repeat("あ", maxHTMLTextBytes) + "</p>"
	if got := len(htmlToText(long)); got > maxHTMLTextBytes+1024 {
		t.Errorf("上限を超えている: %d バイト", got)
	}
}
