package main

import (
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// Gemini の返答は safeHtmlItem[].html に HTML で入っている。
// chatview は本文を RenderText（コードフェンスとインラインコードだけ描き、他はエスケープ）に
// 通すので、HTML をそのまま渡すとタグが文字として並ぶ。ここで markdown 風の平文へ写す。
// 同梱の会話系プラグインはどれも本文が markdown の平文なので、見た目も揃う。
//
// 実物の書き出しに出てくるタグは p / strong / li / code / ul / ol / h2-h4 / table /
// thead / tbody / tr / th / td / pre / a / em / br / hr / blockquote だけで、
// 属性は href と align だけだった。知らないタグは中身のテキストだけを拾う。
//
// 写し方を変えたら cacheSchemaVersion を上げること（取り込み時に写すので、
// 上げないと古いキャッシュが古い写し方のまま残る）。

// maxHTMLTextBytes は1つの返答から取り出す本文の上限。
// 壊れた入力で無限に伸びないようにするための歯止め（実物は数千文字どまり）。
const maxHTMLTextBytes = 1 << 20

// htmlToText は返答の HTML を markdown 風の平文にする。
func htmlToText(source string) string {
	writer := &textWriter{}
	tokenizer := html.NewTokenizer(strings.NewReader(source))

	// listStack は入れ子のリスト。ol なら採番する。
	type listState struct {
		ordered bool
		index   int
	}
	var listStack []listState
	inPre := false
	linkHref := ""
	linkStart := -1

	for !writer.full() {
		switch tokenizer.Next() {
		case html.ErrorToken:
			// io.EOF でも壊れた断片でも、そこまでに拾えたぶんを返す。
			if tokenizer.Err() != io.EOF {
				return writer.result()
			}
			return writer.result()

		case html.TextToken:
			writer.writeText(string(tokenizer.Text()), inPre)

		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := tokenizer.TagName()
			switch tag := string(name); tag {
			case "br":
				writer.newline()
			case "hr":
				writer.blankLine()
				writer.writeRaw("---")
				writer.blankLine()
			case "p", "div", "section":
				writer.blankLine()
			case "h1", "h2", "h3", "h4", "h5", "h6":
				writer.blankLine()
				writer.writeOpen(strings.Repeat("#", headingLevel(tag)) + " ")
			case "ul", "ol":
				listStack = append(listStack, listState{ordered: tag == "ol"})
				writer.newline()
			case "li":
				writer.newline()
				depth := len(listStack)
				prefix := "- "
				if depth > 0 {
					top := &listStack[depth-1]
					if top.ordered {
						top.index++
						prefix = strconv.Itoa(top.index) + ". "
					}
					writer.writeRaw(strings.Repeat("  ", depth-1))
				}
				writer.writeOpen(prefix)
			case "strong", "b":
				writer.writeOpen("**")
			case "em", "i":
				writer.writeOpen("*")
			case "pre":
				inPre = true
				writer.blankLine()
				writer.writeRaw("```")
				writer.newline()
			case "code":
				if !inPre {
					writer.writeOpen("`")
				}
			case "blockquote":
				writer.blankLine()
				writer.writeOpen("> ")
			case "table", "thead", "tbody":
				writer.newline()
			case "tr":
				writer.newline()
				writer.writeRaw("|")
			case "th", "td":
				writer.writeRaw(" ")
			case "a":
				linkHref = attrValue(tokenizer, hasAttr, "href")
				linkStart = writer.length()
			}

		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			switch tag := string(name); tag {
			case "p", "div", "section":
				writer.blankLine()
			case "h1", "h2", "h3", "h4", "h5", "h6":
				writer.blankLine()
			case "ul", "ol":
				if len(listStack) > 0 {
					listStack = listStack[:len(listStack)-1]
				}
				writer.newline()
			case "li":
				writer.newline()
			case "strong", "b":
				writer.writeRaw("**")
			case "em", "i":
				writer.writeRaw("*")
			case "pre":
				inPre = false
				writer.newline()
				writer.writeRaw("```")
				writer.blankLine()
			case "code":
				if !inPre {
					writer.writeRaw("`")
				}
			case "blockquote":
				writer.blankLine()
			case "table":
				writer.blankLine()
			case "tr":
				writer.newline()
			case "th", "td":
				writer.writeRaw(" |")
			case "a":
				// リンク先が本文に出ていないときだけ、末尾に括弧で添える。
				if linkHref != "" && linkStart >= 0 && !strings.Contains(writer.since(linkStart), linkHref) {
					writer.writeRaw(" (" + linkHref + ")")
				}
				linkHref = ""
				linkStart = -1
			}
		}
	}
	return writer.result()
}

// headingLevel は h1-h6 の数字を返す。
func headingLevel(tag string) int {
	if len(tag) == 2 && tag[1] >= '1' && tag[1] <= '6' {
		return int(tag[1] - '0')
	}
	return 3
}

// attrValue は開始タグから属性値を取り出す。
func attrValue(tokenizer *html.Tokenizer, hasAttr bool, want string) string {
	for hasAttr {
		var key, value []byte
		key, value, hasAttr = tokenizer.TagAttr()
		if strings.EqualFold(string(key), want) {
			return string(value)
		}
	}
	return ""
}

// textWriter は markdown 風の平文を組み立てる。
// pre の外では連続した空白を1つに畳み、行頭の空白と3行以上の空行は作らない。
type textWriter struct {
	builder strings.Builder
	// afterOpen は直前に書いたのが開き記号（`**` や `- ` など）かどうか。
	// 開き記号の直後の空白は落とす（`** 太字**` にしないため）が、
	// 閉じ記号の直後の空白は残す（`**太字** のあと` が詰まらないため）。
	afterOpen bool
}

func (w *textWriter) length() int { return w.builder.Len() }

// full は上限に達したかどうか。
func (w *textWriter) full() bool { return w.builder.Len() >= maxHTMLTextBytes }

func (w *textWriter) since(start int) string {
	current := w.builder.String()
	if start < 0 || start > len(current) {
		return ""
	}
	return current[start:]
}

func (w *textWriter) lastByte() byte {
	current := w.builder.String()
	if len(current) == 0 {
		return 0
	}
	return current[len(current)-1]
}

// writeRaw はそのまま足す（閉じ記号・区切りの挿入用）。
func (w *textWriter) writeRaw(s string) {
	w.builder.WriteString(s)
	w.afterOpen = false
}

// writeOpen は開き記号を足す。直後の空白は落とされる。
func (w *textWriter) writeOpen(s string) {
	w.builder.WriteString(s)
	w.afterOpen = true
}

// newline は改行を足す。すでに行頭なら足さない。
func (w *textWriter) newline() {
	if w.builder.Len() == 0 || w.lastByte() == '\n' {
		return
	}
	w.builder.WriteByte('\n')
	w.afterOpen = false
}

// blankLine は空行を作る。すでに空行なら足さない。
func (w *textWriter) blankLine() {
	if w.builder.Len() == 0 {
		return
	}
	w.newline()
	current := w.builder.String()
	if len(current) >= 2 && current[len(current)-2] == '\n' {
		return
	}
	w.builder.WriteByte('\n')
	w.afterOpen = false
}

// writeText は本文を足す。pre の中では原文のまま、外では空白を畳む。
func (w *textWriter) writeText(text string, inPre bool) {
	if text == "" {
		return
	}
	if inPre {
		w.builder.WriteString(w.fit(text))
		w.afterOpen = false
		return
	}

	collapsed := collapseSpaces(text)
	if collapsed == "" {
		return
	}
	// 開き記号の直後・行頭・すでに空白のあとでは、先頭の空白を落とす。
	if strings.HasPrefix(collapsed, " ") {
		last := w.lastByte()
		if w.afterOpen || w.builder.Len() == 0 || last == '\n' || last == ' ' {
			collapsed = collapsed[1:]
		}
	}
	if collapsed == "" {
		return
	}
	w.builder.WriteString(w.fit(collapsed))
	w.afterOpen = false
}

// fit は上限に収まるところまで切る。UTF-8 の途中では切らない。
func (w *textWriter) fit(s string) string {
	room := maxHTMLTextBytes - w.builder.Len()
	if room <= 0 {
		return ""
	}
	if len(s) <= room {
		return s
	}
	cut := s[:room]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}

// collapseSpaces は連続した空白を1つに畳む。先頭と末尾の空白は1つだけ残す。
func collapseSpaces(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	pending := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			pending = true
			continue
		}
		if pending {
			out.WriteByte(' ')
			pending = false
		}
		out.WriteRune(r)
	}
	if pending {
		out.WriteByte(' ')
	}
	return out.String()
}

// result は組み立てた平文を整えて返す。
func (w *textWriter) result() string {
	out := w.builder.String()
	// 行末の空白を落とし、3行以上の空行を2行に畳む。
	lines := strings.Split(out, "\n")
	cleaned := make([]string, 0, len(lines))
	blanks := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t\r")
		if line == "" {
			blanks++
			if blanks > 1 {
				continue
			}
		} else {
			blanks = 0
		}
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
