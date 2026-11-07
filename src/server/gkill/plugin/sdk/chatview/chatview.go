// Package chatview は AI チャット系プラグイン（ChatGPT / Claude.ai / Claude Code / Codex）の
// Kyou 本文 HTML を1か所で描く。
//
// プラグインは自分のログの形を Message へ写すだけにし、head（CSS 変数・テーマ追従・iframe の
// サイズ通知）・吹き出し・チップ・折りたたみ・「見つかりません」の体裁はここで揃える。
// 4本が別々に head をコピーして育った結果、チップの区切り空白の有無や未スタイルの not-found ページ、
// 送信者ラベルの違いといった差が積もっていた（ADR-0315）。
//
// 返事の組み方は ADR-0314（Codex）に揃える: 最終回答と質問を Body に、途中報告・thinking・ツール・
// 変更ファイル・計画は Process に入れると「🛠 作業の経過」1つに時系列で畳まれる。
// rykv は一覧の行にこの HTML をそのまま描くので、経過を開いたまま並べると会話が読めなくなる。
//
// 守ること:
//   - 出力は完結した HTML 文書（iframe は allow-same-origin なしで動くので、外部 CSS / JS は読めない）
//   - postMessage のキーは gkill_iframe_size と gkill_theme だけ。足すときはクライアント
//     （plugin-html-view.vue）と cross-boundary-map.md §9 も同時に変える
//   - dblclick の転送スクリプトは書かない（クライアントが本文の末尾に足す）
//   - Markdown はコードフェンスとインラインコードだけを描く（RenderText）。他はエスケープした
//     プレーンテキストを pre-wrap で出す
//   - ホスト（kyou-view.vue）が関連時刻・rep 名・タグ・テキストを iframe の外に描くので、ここでは繰り返さない
package chatview

import (
	"fmt"
	"html"
	"slices"
	"strings"
	"time"
)

// MaxHTMLBytes は1Kyou の HTML の上限。
// gkill 側の読み取りバッファは 32MB だが、そこまで大きいものは表示できないので手前で打ち切る。
// 超えたところで truncatedNotice を出して残りを捨てる。
const MaxHTMLBytes = 4 * 1024 * 1024

// truncatedNotice は上限で打ち切ったときの断り。
const truncatedNotice = `<div class="truncated">(以降は長いため省略しました)</div>`

// Role は吹き出しの主。背景色とクラス名（msg human / msg assistant）を決める。
type Role string

const (
	RoleHuman     Role = "human"
	RoleAssistant Role = "assistant"
)

// Message は1Kyou ぶんの描画モデル。プラグインは自分のログの形をこれへ写す。
type Message struct {
	// Title は会話 / セッション名。空なら出さない。
	Title string
	// Chips は本文の上に並ぶ短い札（project・branch・model・所要時間の順が目安）。
	// 空は飛ばし、重複は落とし、空白1つで区切る（MCP の HTML→テキスト変換は <span> を改行にしないので、
	// 詰めて並べるとプロジェクト名とブランチ名が1語に連結される）。
	Chips []string
	// Role は吹き出しの主。
	Role Role
	// Sender は吹き出しのラベル。「あなた」「ChatGPT」「Claude」「Claude Code」「Codex」「あなた（質問への回答）」。
	// 質問ブロックの見出し「❓ {Sender} からの質問」にも使う。
	Sender string
	// Text は本文。RenderText を通す（コードフェンスとインラインコードだけ描き、他はエスケープ）。
	Text string
	// Context は本文の直後・Body の前に畳む前置き（Codex の IDE / ブラウザのコンテキストなど）。
	// 本文の前に出すと、一覧のどの行も前置きで埋まって読めなくなる。
	Context []Block
	// Body は返事の本文ブロック（最終回答・質問）。人間の発言では空。
	Body []Block
	// Process は「🛠 作業の経過」1つに時系列で畳むブロック（途中報告・thinking・ツール・変更ファイル・計画）。
	Process []Block
	// RelatedTime は吹き出しの末尾に出す時刻。ゼロなら出さない。
	RelatedTime time.Time
}

// Kind はブロックの種類。
type Kind string

const (
	// KindText は文章。Body では最終回答（.answer）、Process では途中報告（.commentary）として描く。
	KindText Kind = "text"
	// KindQuestion は AI から人間への質問。「❓ {Sender} からの質問」の見出し付き。
	KindQuestion Kind = "question"
	// KindNotice は1行の注記（システム通知・中断・圧縮など）。
	KindNotice Kind = "notice"
	// KindThinking は思考。「💭 thinking (n)」に畳む。
	KindThinking Kind = "thinking"
	// KindTools は連続したツール呼び出し。「🔧 Name ×n」に畳む。サブエージェントを起こした呼び出しはその中にさらに畳む。
	KindTools Kind = "tools"
	// KindPlan は計画。「📋 計画」に畳む。
	KindPlan Kind = "plan"
	// KindFiles は変更したファイルの一覧。「📝 変更したファイル (n)」に畳む。
	KindFiles Kind = "files"
	// KindAgent はツール呼び出しに紐づかないサブエージェントの会話。「🤖 Label」に畳む。
	KindAgent Kind = "agent"
	// KindFold は汎用の折りたたみ。見出し（Summary）はプラグインが絵文字込みで決める。本文（Text）と箇条書き（Items）を持てる。
	KindFold Kind = "fold"
)

// Block は本文・前置き・作業の経過を組み立てる1要素。Kind に応じた欄だけを見る。
type Block struct {
	Kind Kind
	// Text は text / question / notice / plan / fold の本文。
	Text string
	// Thinking は thinking の各断片。
	Thinking []string
	// Tools は tools の各呼び出し。
	Tools []ToolCall
	// Files は files の各ファイル。
	Files []File
	// Agent は agent のサブエージェント。
	Agent *SubAgent
	// Summary は fold の見出し。
	Summary string
	// Items は fold の箇条書き。
	Items []string
}

// ToolCall はツール呼び出し1件。実行結果は持たない（ログの大半を占めるので保存しない）。
type ToolCall struct {
	// Name はツール名。空なら「(不明)」と数える。
	Name string
	// Summary は入力の要約（コマンド・パスなど）。
	Summary string
	// Agent はこの呼び出しで起きたサブエージェントの会話。
	Agent *SubAgent
}

// File は変更したファイル1件。unified diff は持たない。
type File struct {
	Path string
	// Type は add / update / delete など。空なら出さない。
	Type    string
	Added   int
	Removed int
}

// SubAgent はサブエージェントの会話。親の返事の中に畳む。
type SubAgent struct {
	// Label は見出し。空なら「Agent」。
	Label string
	// Prompt はサブエージェントへの指示。
	Prompt string
	// Items はサブエージェントの会話。text / plan / question は本文、tools / files / notice / agent / fold は1行の注記にし、thinking は出さない。
	Items []Block
}

// Render は Message を完結した HTML 文書にする。
func Render(m Message) string {
	r := &renderer{sender: m.Sender}
	r.sb.WriteString(headHTML)

	if m.Title != "" {
		r.sb.WriteString(`<div class="conv-title">`)
		r.sb.WriteString(html.EscapeString(m.Title))
		r.sb.WriteString(`</div>`)
	}
	r.chips(m.Chips)

	class := "assistant"
	if m.Role == RoleHuman {
		class = "human"
	}
	r.sb.WriteString(`<div class="msg `)
	r.sb.WriteString(class)
	r.sb.WriteString(`"><div class="sender">`)
	r.sb.WriteString(html.EscapeString(m.Sender))
	r.sb.WriteString(`</div>`)
	r.sb.WriteString(RenderText(m.Text))

	r.blocks(m.Context, false)
	r.blocks(m.Body, false)
	r.process(m.Process)

	ts := ""
	if !m.RelatedTime.IsZero() {
		ts = m.RelatedTime.Local().Format("2006-01-02 15:04")
	}
	r.sb.WriteString(`<div class="ts">`)
	r.sb.WriteString(html.EscapeString(ts))
	r.sb.WriteString(`</div></div>`)

	r.sb.WriteString(`</body></html>`)
	return r.sb.String()
}

// RenderNotFound は対象の Kyou が見つからなかったときの HTML。head を共有するのでテーマにも追従する。
func RenderNotFound() string {
	return headHTML + `<p>見つかりません</p></body></html>`
}

// FormatDuration は所要時間を「所要 3分12秒」の形にする。1秒未満は出さない（空文字）。
func FormatDuration(d time.Duration) string {
	seconds := int64(d / time.Second)
	if seconds < 1 {
		return ""
	}
	if seconds < 60 {
		return fmt.Sprintf("所要 %d秒", seconds)
	}
	return fmt.Sprintf("所要 %d分%d秒", seconds/60, seconds%60)
}

// SummaryLine はブロック列に含まれるツール呼び出し回数・thinking 件数・変更ファイル数の1行サマリ（エスケープ済み）。
// ツールは回数の多い順（同数なら名前順）に並べる。数えるものが無ければ空文字。
// サブエージェントの中は数えない。
func SummaryLine(blocks []Block) string {
	counts := map[string]int{}
	var order []string
	thinking := 0
	files := 0
	for _, b := range blocks {
		switch b.Kind {
		case KindTools:
			for _, tc := range b.Tools {
				name := toolName(tc)
				if _, ok := counts[name]; !ok {
					order = append(order, name)
				}
				counts[name]++
			}
		case KindThinking:
			thinking += len(b.Thinking)
		case KindFiles:
			files += len(b.Files)
		}
	}
	if len(order) == 0 && thinking == 0 && files == 0 {
		return ""
	}
	slices.SortFunc(order, func(a, b string) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		return strings.Compare(a, b)
	})

	var parts []string
	for _, name := range order {
		parts = append(parts, fmt.Sprintf("%s ×%d", html.EscapeString(name), counts[name]))
	}
	if thinking > 0 {
		parts = append(parts, fmt.Sprintf("thinking ×%d", thinking))
	}
	if files > 0 {
		parts = append(parts, fmt.Sprintf("変更 %dファイル", files))
	}
	return strings.Join(parts, " · ")
}

// renderer は1文書ぶんの組み立て状態。
type renderer struct {
	sb     strings.Builder
	sender string
}

// overflowed は上限を超えていれば断りを書いて真を返す。ブロックを描く直前に呼ぶ。
func (r *renderer) overflowed() bool {
	if r.sb.Len() <= MaxHTMLBytes {
		return false
	}
	r.sb.WriteString(truncatedNotice)
	return true
}

func (r *renderer) div(class string, inner string) {
	r.sb.WriteString(`<div class="`)
	r.sb.WriteString(class)
	r.sb.WriteString(`">`)
	r.sb.WriteString(inner)
	r.sb.WriteString(`</div>`)
}

// chips はチップを空白1つで区切って並べる。空は飛ばし、重複は落とす。
func (r *renderer) chips(chips []string) {
	var shown []string
	for _, c := range chips {
		if c == "" || slices.Contains(shown, c) {
			continue
		}
		shown = append(shown, c)
	}
	if len(shown) == 0 {
		return
	}
	r.sb.WriteString(`<div class="chips">`)
	for i, c := range shown {
		if i > 0 {
			r.sb.WriteString(` `)
		}
		r.sb.WriteString(`<span class="chip">`)
		r.sb.WriteString(html.EscapeString(c))
		r.sb.WriteString(`</span>`)
	}
	r.sb.WriteString(`</div>`)
}

// blocks はブロック列を順に描く。上限を超えたら断りを書いて止める。
func (r *renderer) blocks(blocks []Block, inProcess bool) {
	for _, b := range blocks {
		if r.overflowed() {
			break
		}
		r.block(b, inProcess)
	}
}

// process は作業の経過を1つの折りたたみに描く。中は時系列のまま。
func (r *renderer) process(blocks []Block) {
	if len(blocks) == 0 {
		return
	}
	r.sb.WriteString(`<details class="process"><summary>🛠 作業の経過`)
	if summary := processSummary(blocks); summary != "" {
		r.sb.WriteString(` — `)
		r.sb.WriteString(summary)
	}
	r.sb.WriteString(`</summary>`)
	r.blocks(blocks, true)
	r.sb.WriteString(`</details>`)
}

// processSummary は作業の経過の見出しに添える1行サマリ（エスケープ済み）。途中報告の数を先頭に足す。
func processSummary(blocks []Block) string {
	commentary := 0
	for _, b := range blocks {
		if b.Kind == KindText {
			commentary++
		}
	}
	var parts []string
	if commentary > 0 {
		parts = append(parts, fmt.Sprintf("途中報告 ×%d", commentary))
	}
	if summary := SummaryLine(blocks); summary != "" {
		parts = append(parts, summary)
	}
	return strings.Join(parts, " · ")
}

// block はブロック1つを描く。inProcess は作業の経過の中かどうか（文章のクラスが変わる）。
func (r *renderer) block(b Block, inProcess bool) {
	switch b.Kind {
	case KindText:
		class := "answer"
		if inProcess {
			class = "commentary"
		}
		r.div(class, RenderText(b.Text))

	case KindQuestion:
		r.sb.WriteString(`<div class="question"><div class="question-label">❓ `)
		r.sb.WriteString(html.EscapeString(r.sender))
		r.sb.WriteString(` からの質問</div>`)
		r.sb.WriteString(RenderText(b.Text))
		r.sb.WriteString(`</div>`)

	case KindNotice:
		r.div("notice", html.EscapeString(b.Text))

	case KindThinking:
		fmt.Fprintf(&r.sb, `<details><summary>💭 thinking (%d)</summary>`, len(b.Thinking))
		for _, th := range b.Thinking {
			r.div("thinking-body", RenderText(th))
		}
		r.sb.WriteString(`</details>`)

	case KindTools:
		r.tools(b.Tools)

	case KindPlan:
		r.sb.WriteString(`<details><summary>📋 計画</summary>`)
		r.div("plan-body", RenderText(b.Text))
		r.sb.WriteString(`</details>`)

	case KindFiles:
		r.files(b.Files)

	case KindAgent:
		if b.Agent != nil {
			r.agent(b.Agent)
		}

	case KindFold:
		r.fold(b)
	}
}

// tools は連続したツール呼び出しを1つの折りたたみにまとめる。
// サブエージェントを起こした呼び出しだけは、その中にさらに会話を畳んで入れる。
func (r *renderer) tools(tools []ToolCall) {
	if len(tools) == 0 {
		return
	}
	r.sb.WriteString(`<details><summary>🔧 `)
	r.sb.WriteString(html.EscapeString(toolGroupLabel(tools)))
	r.sb.WriteString(`</summary><ul class="tool-list">`)
	for _, tc := range tools {
		r.sb.WriteString(`<li><span class="tool-name">`)
		r.sb.WriteString(html.EscapeString(tc.Name))
		r.sb.WriteString(`</span>`)
		if tc.Summary != "" {
			r.sb.WriteString(` `)
			r.sb.WriteString(html.EscapeString(tc.Summary))
		}
		r.sb.WriteString(`</li>`)
	}
	r.sb.WriteString(`</ul>`)
	for _, tc := range tools {
		if tc.Agent != nil {
			r.agent(tc.Agent)
		}
	}
	r.sb.WriteString(`</details>`)
}

// toolGroupLabel は "exec ×12  apply_patch ×3" のようなラベル（エスケープ前）。出現順に並べる。
func toolGroupLabel(tools []ToolCall) string {
	counts := map[string]int{}
	var order []string
	for _, tc := range tools {
		name := toolName(tc)
		if _, ok := counts[name]; !ok {
			order = append(order, name)
		}
		counts[name]++
	}
	var parts []string
	for _, name := range order {
		parts = append(parts, fmt.Sprintf("%s ×%d", name, counts[name]))
	}
	return strings.Join(parts, "  ")
}

func toolName(tc ToolCall) string {
	if tc.Name == "" {
		return "(不明)"
	}
	return tc.Name
}

// files は変更したファイルの一覧を描く。
func (r *renderer) files(files []File) {
	if len(files) == 0 {
		return
	}
	fmt.Fprintf(&r.sb, `<details><summary>📝 変更したファイル (%d)</summary><ul class="patch-list">`, len(files))
	for _, f := range files {
		r.sb.WriteString(`<li>`)
		r.sb.WriteString(html.EscapeString(f.Path))
		if f.Type != "" {
			r.sb.WriteString(` (`)
			r.sb.WriteString(html.EscapeString(f.Type))
			r.sb.WriteString(`)`)
		}
		if f.Added > 0 {
			fmt.Fprintf(&r.sb, ` <span class="patch-add">+%d</span>`, f.Added)
		}
		if f.Removed > 0 {
			fmt.Fprintf(&r.sb, ` <span class="patch-del">-%d</span>`, f.Removed)
		}
		r.sb.WriteString(`</li>`)
	}
	r.sb.WriteString(`</ul></details>`)
}

// agent はサブエージェントの会話を折りたたみで描く。
// 中のツール・変更ファイルは1行の注記にし、thinking は出さない。入れ子のサブエージェントも注記で止める（再帰しない）。
func (r *renderer) agent(a *SubAgent) {
	r.sb.WriteString(`<details><summary>🤖 `)
	r.sb.WriteString(html.EscapeString(agentLabel(a)))
	r.sb.WriteString(`</summary>`)
	if a.Prompt != "" {
		r.div("agent-prompt", RenderText(a.Prompt))
	}
	for _, item := range a.Items {
		if r.overflowed() {
			break
		}
		switch item.Kind {
		case KindText, KindPlan, KindQuestion:
			r.div("agent-body", RenderText(item.Text))
		case KindTools:
			r.div("notice", html.EscapeString("🔧 "+toolGroupLabel(item.Tools)))
		case KindFiles:
			r.div("notice", html.EscapeString(fmt.Sprintf("📝 変更 %dファイル", len(item.Files))))
		case KindNotice:
			r.div("notice", html.EscapeString(item.Text))
		case KindAgent:
			if item.Agent != nil {
				r.div("notice", html.EscapeString("🤖 "+agentLabel(item.Agent)))
			}
		case KindFold:
			r.div("notice", html.EscapeString(item.Summary))
		}
	}
	r.sb.WriteString(`</details>`)
}

func agentLabel(a *SubAgent) string {
	if a.Label == "" {
		return "Agent"
	}
	return a.Label
}

// fold は汎用の折りたたみを描く。見出しも本文も箇条書きもプラグインが決めた文字列をそのまま（エスケープして）出す。
func (r *renderer) fold(b Block) {
	if b.Summary == "" && b.Text == "" && len(b.Items) == 0 {
		return
	}
	r.sb.WriteString(`<details><summary>`)
	r.sb.WriteString(html.EscapeString(b.Summary))
	r.sb.WriteString(`</summary>`)
	if b.Text != "" {
		r.div("plan-body", html.EscapeString(b.Text))
	}
	if len(b.Items) > 0 {
		r.sb.WriteString(`<ul class="fold-list">`)
		for _, item := range b.Items {
			r.sb.WriteString(`<li>`)
			r.sb.WriteString(html.EscapeString(item))
			r.sb.WriteString(`</li>`)
		}
		r.sb.WriteString(`</ul>`)
	}
	r.sb.WriteString(`</details>`)
}
