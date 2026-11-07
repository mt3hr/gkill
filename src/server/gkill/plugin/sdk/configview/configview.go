// Package configview は同梱プラグインの設定画面 HTML（GetConfigHTML の応答）を1か所で描く。
//
// プラグインの html.go は自分の文言と統計を Page へ写すだけにし、head（CSS 変数・明暗の追従）・
// 読み込み状況の帯・取り込みの進捗・統計表・警告・指定一覧・展開結果・編集フォームと保存スクリプト・
// config.json を直接編集する案内の体裁はここで揃える。7本が別々に head と保存スクリプトをコピーして育った結果、
// Claude.ai 版が未定義の CSS 変数を参照して入力欄が透明になり、ChatGPT 版は textarea / button の規則を
// 持たないまま配られていた（ADR-0316）。
//
// 守ること:
//   - 出力は完結した HTML 文書（iframe は allow-same-origin なしで動くので、外部 CSS / JS は読めない）
//   - 文言の欄は全部 Prose を通す。描くのは `code` と **強調** だけで、他はエスケープする。
//     HTML を直接受け取る欄は置かない（データが同じ欄に流れたときにエスケープされない）
//   - postMessage のキーは gkill_theme（親 → iframe）・gkill_plugin_config（iframe → 親）・
//     gkill_plugin_config_result（親 → iframe）の3つだけ。足すときはクライアント（use-plugin-config-dialog.ts）
//     と cross-boundary-map.md §9 も同時に変える
//   - ここで描くのは渡された統計だけ。GetConfigHTML は IsAlive（5秒）と同じスロットに並ぶので、
//     呼び出し側はファイルを走査せずキャッシュから即答した値を渡すこと
//   - 明暗は OS の配色（prefers-color-scheme）に追従し、親から gkill_theme が届けばそちらが勝つ
package configview

import (
	"fmt"
	"html"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// DefaultMaxItems は一覧（展開結果・警告の箇条書き・追加の節）に並べる既定の最大件数。超えたぶんは「ほか N 件」。
const DefaultMaxItems = 20

// Page は設定画面1枚ぶんの描画モデル。欄の順に上から描く。
type Page struct {
	// Title は見出し（h2）。
	Title string
	// Intro は見出しの直下の補足（Prose 記法）。空なら出さない。
	Intro string
	// Status は読み込み状況の帯（読み込み済みなら緑、未読み込みなら橙に手順）。nil なら出さない。
	Status *Status
	// Progress は取り込みの進捗。State が空 / idle / error なら出さない。
	Progress Progress
	// Stats は統計表（ラベル・値の2列）。
	Stats []Row
	// Notices は警告。欄が全部空の Notice は飛ばすので、条件付きの警告は空のまま並べてよい。
	Notices []Notice
	// Patterns は「設定されている指定」。空なら節を出さない。ワイルドカードを含む指定には「パターン」の札を付ける。
	Patterns []string
	// Expanded は「展開結果」（指定を実際に展開したフォルダとファイル）。nil なら出さない。
	Expanded *Expanded
	// Sections は追加の節（一覧か表）。フォームの前に並ぶ。
	Sections []Section
	// Form は編集フォーム。nil なら出さない。
	Form *Form
	// ConfigFile は config.json を直接編集する案内（折りたたみ）。nil なら出さない。
	ConfigFile *ConfigFile
}

// Status は読み込み状況の帯。
type Status struct {
	// Loaded が真なら緑の帯（✓ 付き）、偽なら橙の帯（見出しを太字に）。
	Loaded bool
	// Headline は1行目（Prose 記法）。「**N 件**を読み込んでいます」「まだ読み込まれていません。」
	Headline string
	// Lines は続く文（Prose 記法）。
	Lines []string
	// Steps は手順（番号付き。Prose 記法）。未読み込みのときのエクスポート手順など。
	Steps []string
}

// Progress は取り込みの進捗。「初回は空が返る」のが仕様なので、進んでいることが見えないと壊れて見える。
type Progress struct {
	// State はビルダの状態。"" / idle / error は出さない（error は Notices で出す）。
	// scanning → 「データソースを走査しています…」、ingesting → 「取り込み中」、folding → 「集計中」、他 → 「処理中…」。
	State string
	// Done / Total は件数。Total が 0 より大きければ「Done / Total Unit (p%)」と棒を出す。
	Done  int
	Total int
	// Unit は件数の単位（会話・セッション・ファイル・リポジトリ）。
	Unit string
	// Note は補足（Prose 記法）。「初回は1〜2分かかることがあります」など。
	Note string
}

// Row は統計表の1行。
type Row struct {
	Label string
	// Value は値。空なら「—」。
	Value string
	// Code が真なら <code> で出す（パス）。
	Code bool
}

// Notice は警告の帯1つ。Title も Text も Items も空なら出さない。
type Notice struct {
	// Title は1行目（太字。Prose 記法）。
	Title string
	// Text は続く文（Prose 記法）。
	Text string
	// Items は箇条書き。Max（0 なら DefaultMaxItems）を超えたぶんは「ほか N 件」。
	Items []Item
	Max   int
}

// Item は箇条書きの1項目。Label・Code・Text・Tag のうち空でないものを空白1つで区切って並べる。
type Item struct {
	// Label は太字の名前。
	Label string
	// Code は <code> で出す部分（パス・指定）。
	Code string
	// Text は説明（Prose 記法）。
	Text string
	// Tag は小さな札（「パターン」）。
	Tag string
}

// Expanded は「展開結果」。指定を展開して得たフォルダとファイル。
type Expanded struct {
	Dirs  []string
	Files []string
}

// Section は追加の節。Items（箇条書き）か Table（表）のどちらかを持つ。
type Section struct {
	// Title は見出し（h3。Collapsed なら <details> の summary）。
	Title string
	// Collapsed が真なら折りたたみで出す（参照用の長い一覧）。
	Collapsed bool
	// Items は箇条書き。
	Items []Item
	// Table は表。Items より優先する。
	Table *Table
	// Max は並べる最大件数。0 なら DefaultMaxItems、負なら制限しない。
	Max int
	// Empty は 0 件のときの文（Prose 記法）。空なら 0 件のとき節ごと出さない。
	Empty string
	// Note は節の末尾の補足（Prose 記法）。
	Note string
}

// Table は表。セルはエスケープしたプレーンテキスト（Prose 記法ではない。データが入るため）。
type Table struct {
	Header []string
	Rows   [][]string
}

// ConfigFile は config.json を直接編集する案内。折りたたみで出す（フォームが主経路）。
type ConfigFile struct {
	// Path は編集するファイルの実パス。
	Path string
	// Sample は記入例（SampleJSON）。
	Sample string
	// Notes はキーの説明（Prose 記法）。source_dirs の説明は SourceDirsNote で組む。
	Notes []string
}

// Render は設定画面の HTML を返す。
func Render(p Page) string {
	var sb strings.Builder
	sb.WriteString(headHTML)
	sb.WriteString(`<h2>` + html.EscapeString(p.Title) + `</h2>`)
	if p.Intro != "" {
		sb.WriteString(`<p class="hint">` + Prose(p.Intro) + `</p>`)
	}
	renderStatus(&sb, p.Status)
	renderProgress(&sb, p.Progress)
	renderStats(&sb, p.Stats)
	for _, n := range p.Notices {
		renderNotice(&sb, n)
	}
	renderPatterns(&sb, p.Patterns)
	renderExpanded(&sb, p.Expanded)
	for _, s := range p.Sections {
		renderSection(&sb, s)
	}
	renderForm(&sb, p.Form)
	renderConfigFile(&sb, p.ConfigFile)
	sb.WriteString(`</body></html>`)
	return sb.String()
}

// MissingPatterns は「何にもマッチしなかった指定」の警告。0件なら空の Notice（Render が飛ばす）。
func MissingPatterns(missing []string) Notice {
	if len(missing) == 0 {
		return Notice{}
	}
	n := Notice{Title: "次の指定は何にもマッチしませんでした:"}
	for _, m := range missing {
		n.Items = append(n.Items, Item{Code: m})
	}
	return n
}

// ErrorNotice は1件のエラー文の警告。message が空なら空の Notice（Render が飛ばす）。
func ErrorNotice(title string, message string) Notice {
	if message == "" {
		return Notice{}
	}
	return Notice{Title: title, Items: []Item{{Code: message}}}
}

func renderStatus(sb *strings.Builder, s *Status) {
	if s == nil {
		return
	}
	class := "warn"
	if s.Loaded {
		class = "ok"
	}
	sb.WriteString(`<div class="` + class + `">`)
	if s.Headline != "" {
		if s.Loaded {
			sb.WriteString(`<p>✓ ` + Prose(s.Headline) + `</p>`)
		} else {
			sb.WriteString(`<p><strong>` + Prose(s.Headline) + `</strong></p>`)
		}
	}
	for _, line := range s.Lines {
		sb.WriteString(`<p>` + Prose(line) + `</p>`)
	}
	if len(s.Steps) > 0 {
		sb.WriteString(`<ol>`)
		for _, step := range s.Steps {
			sb.WriteString(`<li>` + Prose(step) + `</li>`)
		}
		sb.WriteString(`</ol>`)
	}
	sb.WriteString(`</div>`)
}

func progressLabel(state string) string {
	switch state {
	case "scanning":
		return "データソースを走査しています…"
	case "ingesting":
		return "取り込み中"
	case "folding":
		return "集計中"
	}
	return "処理中…"
}

func renderProgress(sb *strings.Builder, p Progress) {
	switch p.State {
	case "", "idle", "error":
		return
	}
	sb.WriteString(`<div class="ok"><p>` + progressLabel(p.State))
	if p.Total > 0 {
		percent := p.Done * 100 / p.Total
		percent = max(0, min(100, percent))
		fmt.Fprintf(sb, ` %s / %s`, Count(p.Done), Count(p.Total))
		if p.Unit != "" {
			sb.WriteString(` ` + html.EscapeString(p.Unit))
		}
		fmt.Fprintf(sb, ` (%d%%)</p>`, percent)
		fmt.Fprintf(sb, `<div class="progress"><div style="width:%d%%"></div></div>`, percent)
	} else {
		sb.WriteString(`</p>`)
	}
	if p.Note != "" {
		sb.WriteString(`<div class="hint">` + Prose(p.Note) + `</div>`)
	}
	sb.WriteString(`</div>`)
}

func renderStats(sb *strings.Builder, rows []Row) {
	if len(rows) == 0 {
		return
	}
	sb.WriteString(`<table>`)
	for _, r := range rows {
		value := "—"
		if r.Value != "" {
			value = html.EscapeString(r.Value)
			if r.Code {
				value = `<code>` + value + `</code>`
			}
		}
		fmt.Fprintf(sb, `<tr><td class="k">%s</td><td>%s</td></tr>`, html.EscapeString(r.Label), value)
	}
	sb.WriteString(`</table>`)
}

func renderNotice(sb *strings.Builder, n Notice) {
	if n.Title == "" && n.Text == "" && len(n.Items) == 0 {
		return
	}
	sb.WriteString(`<div class="warn">`)
	if n.Title != "" {
		sb.WriteString(`<p><strong>` + Prose(n.Title) + `</strong></p>`)
	}
	if n.Text != "" {
		sb.WriteString(`<p>` + Prose(n.Text) + `</p>`)
	}
	if len(n.Items) > 0 {
		writeItems(sb, n.Items, n.Max)
	}
	sb.WriteString(`</div>`)
}

func renderPatterns(sb *strings.Builder, patterns []string) {
	if len(patterns) == 0 {
		return
	}
	sb.WriteString(`<h3>設定されている指定</h3><ul>`)
	for _, p := range patterns {
		it := Item{Code: p}
		if sdk.HasGlobMeta(p) {
			it.Tag = "パターン"
		}
		writeItem(sb, it)
	}
	sb.WriteString(`</ul>`)
}

func renderExpanded(sb *strings.Builder, e *Expanded) {
	if e == nil {
		return
	}
	fmt.Fprintf(sb, `<h3>展開結果 (フォルダ %s / ファイル %s)</h3>`, Count(len(e.Dirs)), Count(len(e.Files)))
	items := make([]Item, 0, len(e.Dirs)+len(e.Files))
	for _, d := range e.Dirs {
		items = append(items, Item{Code: d, Text: "(再帰的に走査)"})
	}
	for _, f := range e.Files {
		items = append(items, Item{Code: f})
	}
	writeItems(sb, items, 0)
}

func renderSection(sb *strings.Builder, s Section) {
	count := len(s.Items)
	if s.Table != nil {
		count = len(s.Table.Rows)
	}
	if count == 0 && s.Empty == "" {
		return
	}
	if s.Collapsed {
		sb.WriteString(`<details><summary>` + html.EscapeString(s.Title) + `</summary>`)
	} else {
		sb.WriteString(`<h3>` + html.EscapeString(s.Title) + `</h3>`)
	}
	switch {
	case count == 0:
		sb.WriteString(`<p class="hint">` + Prose(s.Empty) + `</p>`)
	case s.Table != nil:
		writeTable(sb, s.Table, s.Max)
	default:
		writeItems(sb, s.Items, s.Max)
	}
	if s.Note != "" {
		sb.WriteString(`<p class="hint">` + Prose(s.Note) + `</p>`)
	}
	if s.Collapsed {
		sb.WriteString(`</details>`)
	}
}

func renderConfigFile(sb *strings.Builder, c *ConfigFile) {
	if c == nil {
		return
	}
	sb.WriteString(`<details><summary>config.json を直接編集する</summary>`)
	sb.WriteString(`<p class="hint">` + Prose("この画面で保存する代わりに、このプラグインのフォルダ(`manifest.json` と同じ場所)の `config.json` を"+
		"直接編集してもかまいません。プラグインの起動時に無ければ自動で作られ、編集は次の検索から反映されます(gkillの再起動は不要)。") + `</p>`)
	if c.Path != "" {
		sb.WriteString(`<p>編集するファイル: <code>` + html.EscapeString(c.Path) + `</code></p>`)
	}
	if c.Sample != "" {
		sb.WriteString(`<pre>` + html.EscapeString(c.Sample) + `</pre>`)
	}
	for _, note := range c.Notes {
		if note == "" {
			continue
		}
		sb.WriteString(`<div class="hint">` + Prose(note) + `</div>`)
	}
	sb.WriteString(`</details>`)
}

// limitOf は Max の規約（0 なら既定、負なら無制限）を件数に直す。
func limitOf(max int, n int) int {
	switch {
	case max < 0:
		return n
	case max == 0:
		return DefaultMaxItems
	}
	return max
}

func writeItems(sb *strings.Builder, items []Item, max int) {
	limit := limitOf(max, len(items))
	sb.WriteString(`<ul>`)
	for i, it := range items {
		if i >= limit {
			fmt.Fprintf(sb, `<li>ほか %s 件</li>`, Count(len(items)-i))
			break
		}
		writeItem(sb, it)
	}
	sb.WriteString(`</ul>`)
}

func writeItem(sb *strings.Builder, it Item) {
	var parts []string
	if it.Label != "" {
		parts = append(parts, `<strong>`+html.EscapeString(it.Label)+`</strong>`)
	}
	if it.Code != "" {
		parts = append(parts, `<code>`+html.EscapeString(it.Code)+`</code>`)
	}
	if it.Text != "" {
		parts = append(parts, Prose(it.Text))
	}
	if it.Tag != "" {
		parts = append(parts, `<span class="tag">`+html.EscapeString(it.Tag)+`</span>`)
	}
	sb.WriteString(`<li>` + strings.Join(parts, " ") + `</li>`)
}

func writeTable(sb *strings.Builder, t *Table, max int) {
	limit := limitOf(max, len(t.Rows))
	sb.WriteString(`<table>`)
	if len(t.Header) > 0 {
		sb.WriteString(`<tr>`)
		for _, h := range t.Header {
			sb.WriteString(`<th>` + html.EscapeString(h) + `</th>`)
		}
		sb.WriteString(`</tr>`)
	}
	shown := 0
	for _, row := range t.Rows {
		if shown >= limit {
			break
		}
		sb.WriteString(`<tr>`)
		for _, cell := range row {
			sb.WriteString(`<td>` + html.EscapeString(cell) + `</td>`)
		}
		sb.WriteString(`</tr>`)
		shown++
	}
	sb.WriteString(`</table>`)
	if len(t.Rows) > shown {
		fmt.Fprintf(sb, `<p class="hint">ほか %s 件</p>`, Count(len(t.Rows)-shown))
	}
}
