package mcp

import (
	"regexp"
	"strings"
	"testing"
)

// 期待値の "バックスラッシュ + u + 4桁" は文字コードから組み立てる（ソース上の \u は編集ツールが畳むことがある）。
var backslashChar = string(rune(92))

// 認可成功ページの redirect_uri は動的登録したクライアントが名乗る値。<script> の中へ JSON 文字列として
// 埋め込むので、"</script>" や U+2028 を含んでも script 要素が閉じたり構文が壊れたりしないこと
// （CodeQL go/reflected-xss の指摘箇所）。普通の URL は & を含めてそのまま（ログイン後の遷移に使う）。
func TestRenderSuccessPageEscapesRedirectURLForScriptContext(t *testing.T) {
	t.Run("a normal URL is embedded verbatim including &", func(t *testing.T) {
		html := RenderSuccessPage("https://client.example/cb?code=abc&state=xyz")
		match := regexp.MustCompile(`window\.location\.href\s*=\s*"([^"]+)"`).FindStringSubmatch(html)
		expectTrue(t, match != nil, "location.href assignment not found")
		expectEqual(t, match[1], "https://client.example/cb?code=abc&state=xyz")
	})

	t.Run("</script> and line separators cannot break out of the script element", func(t *testing.T) {
		lineSeparator := string(rune(0x2028))
		hostile := "https://client.example/cb?x=</script><script>alert(1)</script>" + lineSeparator + "y"
		html := RenderSuccessPage(hostile)
		expectTrue(t, !strings.Contains(html, "</script><script>alert"), "raw </script> reached the page")
		expectTrue(t, strings.Contains(html, backslashChar+"u003c/script"+backslashChar+"u003e"), "< > not escaped: %s", html)
		expectTrue(t, !strings.Contains(html, lineSeparator), "raw U+2028 reached the page")
		expectTrue(t, strings.Contains(html, backslashChar+"u2028"), "U+2028 not escaped")
		// script 要素はテンプレートの1つだけが閉じる
		expectEqual(t, strings.Count(html, "</script>"), 1)
	})

	t.Run("jsStringLiteral keeps JSON escapes for quotes and backslashes", func(t *testing.T) {
		expectEqual(t, jsStringLiteral(`a"b`+backslashChar+`c`), `"a`+backslashChar+`"b`+backslashChar+backslashChar+`c"`)
		expectEqual(t, jsStringLiteral("<>"), `"`+backslashChar+"u003c"+backslashChar+`u003e"`)
	})
}

// ログインページの利用者由来の値（client_name / error / redirect_uri など）は HTML としてエスケープされる。
// テンプレート自身の <script> 要素は残るので、利用者由来の断片だけを見る。
func TestRenderLoginPageEscapesClientProvidedValues(t *testing.T) {
	html := RenderLoginPage(LoginPageParams{
		ClientID:    "cid",
		ClientName:  `<b onmouseover="x">Evil & Co</b>`,
		RedirectURI: `https://client.example/cb?"><script>alert(2)`,
		State:       "st",
		Scope:       "gkill:read",
		Error:       "<img src=x onerror=alert(1)>",
	})
	for _, raw := range []string{`<b onmouseover=`, `"><script>alert(2)`, `<img src=x`} {
		expectTrue(t, !strings.Contains(html, raw), "raw %q reached the login page", raw)
	}
	for _, escaped := range []string{`&lt;b onmouseover=&quot;x&quot;&gt;Evil &amp; Co&lt;/b&gt;`, `&quot;&gt;&lt;script&gt;alert(2)`, `&lt;img src=x onerror=alert(1)&gt;`} {
		expectTrue(t, strings.Contains(html, escaped), "escaped form %q missing", escaped)
	}
	expectEqual(t, escHtml(`&<>"`), `&amp;&lt;&gt;&quot;`)
}
