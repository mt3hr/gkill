package mcp

// html_text.go — プラグインのコンテンツHTMLをAI向けのプレーンテキストに落とす変換。

import "testing"

func TestDecodeHtmlEntities(t *testing.T) {
	t.Run("decodes named entities", func(t *testing.T) {
		expectEqual(t, DecodeHtmlEntities("a &lt;b&gt; c &quot;d&quot;"), `a <b> c "d"`)
	})

	t.Run("decodes numeric and hex entities", func(t *testing.T) {
		expectEqual(t, DecodeHtmlEntities("&#65;&#x42;"), "AB")
	})

	t.Run("decodes &amp; last so escaped entities survive", func(t *testing.T) {
		expectEqual(t, DecodeHtmlEntities("&amp;lt;script&amp;gt;"), "&lt;script&gt;")
	})

	t.Run("decodes nbsp to a normal space", func(t *testing.T) {
		expectEqual(t, DecodeHtmlEntities("a&nbsp;b"), "a b")
	})

	t.Run("leaves unknown entities untouched", func(t *testing.T) {
		expectEqual(t, DecodeHtmlEntities("&notarealentity; x"), "&notarealentity; x")
	})

	t.Run("drops out-of-range code points instead of throwing", func(t *testing.T) {
		expectEqual(t, DecodeHtmlEntities("&#x999999999;"), "")
	})
}

func TestHtmlToText(t *testing.T) {
	text := func(html any) string { return HtmlToText(html, 0).Text }

	t.Run("returns empty result for empty or non-string input", func(t *testing.T) {
		expectEqual(t, htmlToTextObj(HtmlToText("", 0)), obj("text", "", "truncated", false))
		expectEqual(t, htmlToTextObj(HtmlToText(nil, 0)), obj("text", "", "truncated", false))
		expectEqual(t, htmlToTextObj(HtmlToText(undefinedValue(), 0)), obj("text", "", "truncated", false))
	})

	t.Run("drops style and script contents", func(t *testing.T) {
		html := "<html><head><style>body { color: red; }</style>" +
			"<script>var x = 1; document.write('nope');</script></head>" +
			"<body><p>本文</p></body></html>"
		got := text(html)
		expectEqual(t, got, "本文")
		mustNotContain(t, got, "color")
		mustNotContain(t, got, "document.write")
	})

	t.Run("drops HTML comments", func(t *testing.T) {
		expectEqual(t, text("<p>a<!-- hidden -->b</p>"), "ab")
	})

	t.Run("drops script contents even with attributes on the end tag", func(t *testing.T) {
		// "</script\t\n bar>" のような終了タグも終了として扱う (CodeQL js/bad-tag-filter)
		got := text("<p>a</p><script>var secret = 1;</script\t\n bar><p>b</p>")
		expectEqual(t, got, "a\nb")
		mustNotContain(t, got, "secret")
	})

	t.Run("does not end script at a longer tag name like </scripter>", func(t *testing.T) {
		expectEqual(t, text("<script>var x = '</scripter>';</script><p>ok</p>"), "ok")
	})

	t.Run("drops unclosed comment, script and style to the end of input", func(t *testing.T) {
		expectEqual(t, text("<p>a</p><!-- never closed <p>hidden</p>"), "a")
		expectEqual(t, text("<p>a</p><script>var leak = 1;"), "a")
		expectEqual(t, text("<p>a</p><style>body { color: red; }"), "a")
	})

	t.Run("drops an unclosed tag instead of leaving it as text", func(t *testing.T) {
		expectEqual(t, text("<p>a</p><span class=x"), "a")
	})

	t.Run("keeps a literal '<' that does not start a tag", func(t *testing.T) {
		expectEqual(t, text("<p>a < b</p>"), "a < b")
	})

	t.Run("removes uppercase and attribute-heavy tags", func(t *testing.T) {
		expectEqual(t, text(`<SCRIPT type="text/javascript">nope();</SCRIPT><P CLASS="x">a</P>`), "a")
	})

	t.Run("drops doctype and processing instructions", func(t *testing.T) {
		expectEqual(t, text(`<!doctype html><?xml version="1.0"?><p>a</p>`), "a")
	})

	t.Run("turns block boundaries and br into newlines", func(t *testing.T) {
		expectEqual(t, text("<div>one</div><div>two<br>three</div>"), "one\ntwo\nthree")
	})

	t.Run("keeps inline tags inline", func(t *testing.T) {
		expectEqual(t, text("<p>a <strong>bold</strong> word</p>"), "a bold word")
	})

	t.Run("keeps details/summary text (plugin collapses tool calls there)", func(t *testing.T) {
		got := text("<details><summary>ツール実行 3件</summary><pre>Read foo.go</pre></details>")
		mustContain(t, got, "ツール実行 3件")
		mustContain(t, got, "Read foo.go")
	})

	t.Run("collapses runs of blank lines into one", func(t *testing.T) {
		expectEqual(t, text("<div>a</div><div></div><div></div><div>b</div>"), "a\nb")
	})

	t.Run("decodes entities in the extracted text", func(t *testing.T) {
		expectEqual(t, text("<p>if a &lt; b &amp;&amp; c &gt; d</p>"), "if a < b && c > d")
	})

	t.Run("does not resurrect escaped markup as tags", func(t *testing.T) {
		// プラグインは本文を html.EscapeString してから埋め込むので、
		// エスケープ済みの <script> はタグ除去より後にデコードされて本文として残る。
		expectEqual(t, text("<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>"), "<script>alert(1)</script>")
	})

	t.Run("truncates at maxLength and reports it", func(t *testing.T) {
		result := HtmlToText("<p>abcdefghij</p>", 4)
		expectTrue(t, result.Truncated, "not truncated")
		expectEqual(t, result.Text, "abcd\n…(truncated)")
	})

	t.Run("does not truncate when under maxLength", func(t *testing.T) {
		result := HtmlToText("<p>abc</p>", 10)
		expectTrue(t, !result.Truncated, "truncated")
		expectEqual(t, result.Text, "abc")
	})

	t.Run("ignores non-positive maxLength", func(t *testing.T) {
		expectTrue(t, !HtmlToText("<p>abc</p>", 0).Truncated, "truncated with maxLength 0")
	})
}
