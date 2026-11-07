package chatview

import (
	"html"
	"regexp"
	"strings"
)

// RenderText は本文を HTML にする。Markdown のうちコードフェンス（``` / ~~~）とインラインコード（`code`）だけを描き、
// それ以外はエスケープしたプレーンテキストのまま返す（吹き出しは white-space: pre-wrap なので改行はそのまま見える）。
//
// 見出し・箇条書き・強調などは描かない。AI の返事は Markdown とは限らず、描くと MCP の HTML→テキスト変換の形も変わるため、
// 利用者が「コードブロックだけ」と決めた（ADR-0315）。
//
// 規則は CommonMark の部分集合:
//   - 開き: 先頭の空白3つ以下 ＋ 同じ文字（` か ~）3つ以上 ＋ 任意の info 文字列。info の先頭語が英数字と _ + . # - だけで
//     32文字以内なら class="lang-<語>" を付ける（検証していない属性値は出さない）。` のフェンスで info に ` があればフェンスにしない
//   - 閉じ: 同じ文字が開きと同数以上で、後ろは空白のみ。中にある短い列は本文
//   - 閉じが無ければ末尾までコード（取り落とさない）
//   - 開き行の直前と閉じ行の直後の改行は1つずつ落とす（pre-wrap でブロックの前後に空行が出ないように）
//   - インラインは N 個の ` で開き、同じ行の次の「ちょうど N 個」で閉じる。無ければそのまま。前後1つずつの空白は両方あるときだけ落とす
//   - \r\n は \n に揃える。` も ~ も無い文字列では（揃えたあとの文字列の）html.EscapeString と同じ結果になる
//
// MCP の HtmlToText は <pre> をブロック要素として前後に改行を入れ、コード内の改行はそのまま残す。
func RenderText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.ContainsAny(s, "`~") {
		return html.EscapeString(s)
	}
	lines := strings.Split(s, "\n")

	var sb strings.Builder
	var prose []string
	// prose の行を "\n" でつなぐ。Split が開き行・閉じ行に隣接する改行を消しているので、
	// これでフェンスの前後の改行がちょうど1つずつ落ちる。
	flushProse := func() {
		for i, line := range prose {
			if i > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(renderInline(line))
		}
		prose = prose[:0]
	}

	for i := 0; i < len(lines); i++ {
		fence, ok := parseFenceOpener(lines[i])
		if !ok {
			prose = append(prose, lines[i])
			continue
		}
		flushProse()

		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if fence.closes(lines[j]) {
				end = j
				break
			}
		}
		sb.WriteString(`<pre><code`)
		if fence.lang != "" {
			sb.WriteString(` class="lang-`)
			sb.WriteString(fence.lang)
			sb.WriteString(`"`)
		}
		sb.WriteString(`>`)
		sb.WriteString(html.EscapeString(strings.Join(lines[i+1:end], "\n")))
		sb.WriteString(`</code></pre>`)
		i = end
	}
	flushProse()
	return sb.String()
}

// fenceOpener は開きフェンスの形。
type fenceOpener struct {
	char   byte
	length int
	lang   string
}

var (
	fenceOpenerRe = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")
	fenceLangRe   = regexp.MustCompile(`^[A-Za-z0-9_+.#-]{1,32}$`)
)

// parseFenceOpener は行が開きフェンスなら、その形を返す。
func parseFenceOpener(line string) (fenceOpener, bool) {
	m := fenceOpenerRe.FindStringSubmatch(line)
	if m == nil {
		return fenceOpener{}, false
	}
	marker, info := m[1], strings.TrimSpace(m[2])
	if marker[0] == '`' && strings.Contains(info, "`") {
		return fenceOpener{}, false
	}
	f := fenceOpener{char: marker[0], length: len(marker)}
	if fields := strings.Fields(info); len(fields) > 0 && fenceLangRe.MatchString(fields[0]) {
		f.lang = fields[0]
	}
	return f, true
}

// closes は行がこのフェンスの閉じかを返す。同じ文字が開きと同数以上で、後ろは空白だけ。
func (f fenceOpener) closes(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return false
	}
	n := 0
	for n < len(trimmed) && trimmed[n] == f.char {
		n++
	}
	if n < f.length {
		return false
	}
	return strings.TrimSpace(trimmed[n:]) == ""
}

// renderInline は1行の中のインラインコードを <code> にし、残りをエスケープする。
func renderInline(line string) string {
	if !strings.Contains(line, "`") {
		return html.EscapeString(line)
	}
	var sb strings.Builder
	i := 0
	for i < len(line) {
		if line[i] != '`' {
			j := strings.IndexByte(line[i:], '`')
			if j < 0 {
				sb.WriteString(html.EscapeString(line[i:]))
				break
			}
			sb.WriteString(html.EscapeString(line[i : i+j]))
			i += j
			continue
		}
		n := backtickRun(line, i)
		closeAt := findBacktickRun(line, i+n, n)
		if closeAt < 0 {
			// 対になる列が無い。この列は文字のまま出して先へ進む（CommonMark と同じ）
			sb.WriteString(line[i : i+n])
			i += n
			continue
		}
		content := line[i+n : closeAt]
		if len(content) >= 2 && content[0] == ' ' && content[len(content)-1] == ' ' && strings.TrimSpace(content) != "" {
			content = content[1 : len(content)-1]
		}
		sb.WriteString(`<code>`)
		sb.WriteString(html.EscapeString(content))
		sb.WriteString(`</code>`)
		i = closeAt + n
	}
	return sb.String()
}

// backtickRun は位置 at から続く ` の個数を返す。
func backtickRun(line string, at int) int {
	n := 0
	for at+n < len(line) && line[at+n] == '`' {
		n++
	}
	return n
}

// findBacktickRun は位置 from 以降で「ちょうど want 個」の ` の列が始まる位置を返す。無ければ -1。
func findBacktickRun(line string, from int, want int) int {
	for i := from; i < len(line); {
		if line[i] != '`' {
			i++
			continue
		}
		n := backtickRun(line, i)
		if n == want {
			return i
		}
		i += n
	}
	return -1
}
