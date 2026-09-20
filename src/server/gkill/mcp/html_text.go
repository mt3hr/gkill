package mcp

// プラグインが返すコンテンツHTMLをAIが読めるプレーンテキストに落とすための変換（旧 html-text.mjs）。
//
// プラグインのコンテンツHTMLは <style> と <script> を含む完結したHTML文書で、
// バイト数の大半が表示用のボイラープレートになる。生HTMLをそのままMCPで返すと
// トークンを浪費するだけなので、既定ではここでテキストに変換して返す。
// 外部依存を増やしたくないので、正規表現ベースの軽量な変換にとどめる。

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

var blockTagSet = NewStringSet(
	"address", "article", "aside", "blockquote", "details", "dd", "div", "dl", "dt", "fieldset",
	"figcaption", "figure", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hr",
	"li", "main", "nav", "ol", "p", "pre", "section", "summary", "table", "tbody", "td", "tfoot",
	"th", "thead", "title", "tr", "ul",
)

// boundary はタグ由来の改行位置を表す内部マーカー。入力に含まれていたものは先に消す。
const boundary = "\x00"

// tagStartRegex はタグ開始の判定（走査位置から直接マッチさせる）。
var tagStartRegex = regexp.MustCompile(`^<(/?)([a-zA-Z][a-zA-Z0-9-]*)`)

var namedEntities = map[string]string{
	"nbsp":   " ",
	"lt":     "<",
	"gt":     ">",
	"quot":   `"`,
	"apos":   "'",
	"#39":    "'",
	"yen":    "¥",
	"copy":   "©",
	"reg":    "®",
	"hellip": "…",
	"mdash":  "—",
	"ndash":  "–",
}

var (
	hexEntityRegex   = regexp.MustCompile(`&#x([0-9a-fA-F]+);`)
	decEntityRegex   = regexp.MustCompile(`&#(\d+);`)
	namedEntityRegex = regexp.MustCompile(`&([a-zA-Z]+);`)
	whitespaceRunRe  = regexp.MustCompile("[ \t　]+")
)

// DecodeHtmlEntities はHTMLエンティティを実文字に戻す。
// &amp; を最後に処理しないと "&amp;lt;" が "<" になってしまうので、順序を守る。
func DecodeHtmlEntities(text string) string {
	out := hexEntityRegex.ReplaceAllStringFunc(text, func(match string) string {
		sub := hexEntityRegex.FindStringSubmatch(match)
		return safeFromCodePoint(sub[1], 16)
	})
	out = decEntityRegex.ReplaceAllStringFunc(out, func(match string) string {
		sub := decEntityRegex.FindStringSubmatch(match)
		return safeFromCodePoint(sub[1], 10)
	})
	out = namedEntityRegex.ReplaceAllStringFunc(out, func(match string) string {
		sub := namedEntityRegex.FindStringSubmatch(match)
		if replacement, ok := namedEntities[strings.ToLower(sub[1])]; ok {
			return replacement
		}
		return match
	})
	return strings.ReplaceAll(out, "&amp;", "&")
}

// safeFromCodePoint は String.fromCodePoint の安全版。範囲外は空文字列。
func safeFromCodePoint(digits string, base int) string {
	n, err := strconv.ParseInt(digits, base, 64)
	if err != nil || n < 0 || n > 0x10ffff {
		return ""
	}
	if n >= 0xD800 && n <= 0xDFFF {
		// JS は孤立サロゲートを持てるが Go の文字列では表せない。空にせず U+FFFD にする
		return "�"
	}
	return string(rune(n))
}

// stripTags はHTMLからタグを取り除き、タグ由来の改行位置を boundary マーカーで
// 表したテキストを返す。コメント・script・style は中身ごと落とす。
//
// 正規表現replaceの繰り返しだと未閉鎖の <!-- や <script>、"</script foo>" 形式の
// 終了タグが取りこぼされて中身が本文に漏れるため、1パスの文字走査で処理する。
// タグ由来の改行に素の "\n" を使わないのは、</div><div> のような隣接だけで
// 空行ができて入れ子の深いHTMLが空行だらけになるため。マーカーの連続は
// 後段でまとめて1つの改行に落とす。
func stripTags(html string) string {
	var out strings.Builder
	i := 0
	n := len(html)
	for i < n {
		ch := html[i]
		if ch == 0 {
			// 入力に混ざっていた内部マーカーはタグ由来の改行と区別できないので捨てる
			i++
			continue
		}
		if ch != '<' {
			out.WriteByte(ch)
			i++
			continue
		}
		if strings.HasPrefix(html[i:], "<!--") {
			commentEnd := strings.Index(html[i+4:], "-->")
			if commentEnd == -1 {
				i = n
			} else {
				i = i + 4 + commentEnd + 3
			}
			continue
		}
		// <!doctype ...> や <?xml ...?> はタグごと捨てる
		if i+1 < n && (html[i+1] == '!' || html[i+1] == '?') {
			declEnd := strings.Index(html[i+2:], ">")
			if declEnd == -1 {
				i = n
			} else {
				i = i + 2 + declEnd + 1
			}
			continue
		}
		tagMatch := tagStartRegex.FindStringSubmatch(html[i:])
		if tagMatch == nil {
			// タグ開始に見えない "<" (例: "a < b") はリテラルとして残す
			out.WriteByte(ch)
			i++
			continue
		}
		isClosing := tagMatch[1] == "/"
		tagName := strings.ToLower(tagMatch[2])
		tagEnd := strings.Index(html[i+len(tagMatch[0]):], ">")
		// ">" の無い未閉鎖タグは末尾まで捨てる
		if tagEnd == -1 {
			i = n
		} else {
			i = i + len(tagMatch[0]) + tagEnd + 1
		}
		if !isClosing && (tagName == "script" || tagName == "style") {
			i = skipRawTextContent(html, i, tagName)
			continue
		}
		if tagName == "br" || blockTagSet.Has(tagName) {
			out.WriteString(boundary)
		}
	}
	return out.String()
}

// skipRawTextContent は <script>/<style> の中身を終了タグごと読み飛ばす。
// from は開始タグの ">" の直後の位置。戻り値は終了タグの ">" の直後（無ければ末尾）。
func skipRawTextContent(html string, from int, tagName string) int {
	lowerHTML := strings.ToLower(html)
	closer := "</" + tagName
	i := from
	for i < len(html) {
		found := strings.Index(lowerHTML[i:], closer)
		if found == -1 {
			return len(html)
		}
		found += i
		afterIndex := found + len(closer)
		// "</scripter>" のような別タグ名は終了タグではないので読み進める
		if afterIndex < len(html) {
			after := html[afterIndex]
			if after != '>' && after != '/' && !isASCIISpace(after) {
				i = afterIndex
				continue
			}
		}
		closeEnd := strings.Index(html[afterIndex:], ">")
		if closeEnd == -1 {
			return len(html)
		}
		return afterIndex + closeEnd + 1
	}
	return len(html)
}

func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}

// HtmlToTextResult は変換結果と、切り詰めが起きたか。
type HtmlToTextResult struct {
	Text      string
	Truncated bool
}

// HtmlToText はHTMLをプレーンテキストに変換する。maxLength が正なら超えた分を切り落として末尾に印を付ける。
// html が文字列でなければ空の結果（旧実装が null / undefined を受けていたのと同じ）。
func HtmlToText(html any, maxLength int) HtmlToTextResult {
	s, ok := html.(string)
	if !ok || s == "" {
		return HtmlToTextResult{Text: "", Truncated: false}
	}

	work := stripTags(s)

	// マーカーで区切り、区切りごとにエンティティを戻す。
	// デコードを分割より後にするのは、&#0; がマーカーと誤認されないようにするため。
	// セグメント内の素の改行は元テキスト由来なので保持し、空行の連続だけまとめる。
	lines := []string{}
	for _, segment := range strings.Split(work, boundary) {
		if segment == "" {
			continue
		}
		for _, rawLine := range strings.Split(DecodeHtmlEntities(segment), "\n") {
			line := jsTrim(whitespaceRunRe.ReplaceAllString(rawLine, " "))
			if line == "" && (len(lines) == 0 || lines[len(lines)-1] == "") {
				continue
			}
			lines = append(lines, line)
		}
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	text := strings.Join(lines, "\n")
	truncated := false
	if maxLength > 0 && jsLength(text) > maxLength {
		text = jsSlice(text, 0, maxLength) + "\n…(truncated)"
		truncated = true
	}
	return HtmlToTextResult{Text: text, Truncated: truncated}
}

// jsLength は JS の String#length（UTF-16 コード単位数）。
func jsLength(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// jsSlice は JS の String#slice(start, end)（UTF-16 コード単位で切る）。
func jsSlice(s string, start, end int) string {
	units := utf16.Encode([]rune(s))
	if start < 0 {
		start = 0
	}
	if end > len(units) {
		end = len(units)
	}
	if start >= end {
		return ""
	}
	return string(utf16.Decode(units[start:end]))
}
