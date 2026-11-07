package chatview

import (
	"html"
	"testing"
)

func TestRenderTextCodeFences(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"素のフェンス", "a\n```\nx\n```\nb", "a<pre><code>x</code></pre>b"},
		{"言語名は class に", "```go\nfmt.Println()\n```", `<pre><code class="lang-go">fmt.Println()</code></pre>`},
		{"info の2語目以降は無視", "```go title=x\ny\n```", `<pre><code class="lang-go">y</code></pre>`},
		{"不正な info は class 無し", "```<script>\ny\n```", `<pre><code>y</code></pre>`},
		{"チルダのフェンス", "~~~\nx\n~~~", "<pre><code>x</code></pre>"},
		{"閉じは開きより長くてもよい", "```\nx\n`````", "<pre><code>x</code></pre>"},
		{"中の短い列は本文", "````\n```\nx\n```\n````", "<pre><code>```\nx\n```</code></pre>"},
		{"閉じが無ければ末尾まで", "a\n```\nx\ny", "a<pre><code>x\ny</code></pre>"},
		{"先頭の空白3つまでは開き", "   ```\nx\n   ```", "<pre><code>x</code></pre>"},
		{"先頭の空白4つは開きでない", "    ```\nx", "    ```\nx"},
		{"CRLF は LF に揃える", "a\r\n```\r\nx\r\n```\r\nb", "a<pre><code>x</code></pre>b"},
		{"中身はエスケープ", "```\n<b>&\n```", "<pre><code>&lt;b&gt;&amp;</code></pre>"},
		{"利用者が置いた空行は1つ残る", "a\n\n```\nx\n```\n\nb", "a\n<pre><code>x</code></pre>\nb"},
		{"行の途中の ``` はフェンスでない", "a ```b\nc", "a ```b\nc"},
		{"info にバッククォートがあれば ` のフェンスでない", "``` a`b\nc", "``` a`b\nc"},
		{"フェンスが連続", "```\nx\n```\n```\ny\n```", "<pre><code>x</code></pre><pre><code>y</code></pre>"},
	}
	for _, c := range cases {
		if got := RenderText(c.in); got != c.want {
			t.Errorf("%s: RenderText(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestRenderTextInlineCode(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"1個", "run `go test` now", "run <code>go test</code> now"},
		{"2個で中に1個を含める", "a ``x ` y`` b", "a <code>x ` y</code> b"},
		{"対が無ければ文字のまま", "a ` b", "a ` b"},
		{"行をまたがない", "a `b\nc` d", "a `b\nc` d"},
		{"前後1つずつの空白は落とす", "` x `", "<code>x</code>"},
		{"片側だけの空白は残す", "` x`", "<code> x</code>"},
		{"空白だけなら落とさない", "`  `", "<code>  </code>"},
		{"中身はエスケープ", "`<a>`", "<code>&lt;a&gt;</code>"},
		{"長さの違う列は閉じにならない", "a `b `` c` d", "a <code>b `` c</code> d"},
	}
	for _, c := range cases {
		if got := RenderText(c.in); got != c.want {
			t.Errorf("%s: RenderText(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestRenderTextPlainEqualsEscapeString(t *testing.T) {
	for _, in := range []string{"", "plain", "a < b && c > d\n\"quoted\" 'single'", "日本語\n\n改行"} {
		if got, want := RenderText(in), html.EscapeString(in); got != want {
			t.Errorf("RenderText(%q) = %q, want %q", in, got, want)
		}
	}
}
