package configview

import (
	"maps"
	"regexp"
	"strings"
	"testing"
)

func TestRenderHeadContract(t *testing.T) {
	// use-plugin-config-dialog.ts との契約。明暗は OS の配色に追従し、親から gkill_theme が届けばそちらが勝つ。
	out := Render(Page{Title: "x"})
	for _, want := range []string{
		`<!DOCTYPE html>`, `<meta charset="utf-8">`,
		`@media (prefers-color-scheme: dark)`, `:root:not([data-theme="light"])`, `:root[data-theme="dark"]`,
		`e.data.gkill_theme`, `setAttribute('data-theme'`,
		// ChatGPT 版に無かった規則。入力欄とボタンの見た目はここで決まる
		"textarea, input[type=text] {", "button {", "button:disabled {",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q が無い", want)
		}
	}
	if !strings.HasSuffix(out, "</body></html>") {
		t.Error("HTML が閉じていない")
	}
	// 本文（chatview）と違い、設定ダイアログは高さを自分で持つ
	if strings.Contains(out, "gkill_iframe_size") || strings.Contains(out, "dblclick") {
		t.Error("設定画面でサイズ通知や dblclick 転送を書いている")
	}
}

func TestRenderEveryCSSVariableIsDefined(t *testing.T) {
	// Claude.ai 版は --field-bg / --btn-bg / --btn-color を参照していたのに定義が無く、入力欄が透明になっていた。
	// 参照している変数が明暗の両方で定義されていることを固定する。
	out := Render(Page{Title: "x"})
	defined := func(block string) map[string]bool {
		m := map[string]bool{}
		for _, match := range regexp.MustCompile(`(--[a-z-]+):`).FindAllStringSubmatch(block, -1) {
			m[match[1]] = true
		}
		return m
	}
	light, dark := defined(lightVars), defined(darkVars)
	if !maps.Equal(light, dark) {
		t.Errorf("明暗で定義している変数が揃っていない: light=%v dark=%v", light, dark)
	}
	for _, match := range regexp.MustCompile(`var\((--[a-z-]+)\)`).FindAllStringSubmatch(out, -1) {
		if !light[match[1]] {
			t.Errorf("参照している CSS 変数 %s が定義されていない", match[1])
		}
	}
}

func TestRenderPostMessageKeys(t *testing.T) {
	// postMessage のキーは3つだけ。足すときはクライアント（use-plugin-config-dialog.ts）と cross-boundary-map.md §9 も変える
	out := Render(Page{Title: "x", Form: &Form{Fields: []Field{{Key: "source_dirs"}}}})
	keys := map[string]bool{}
	for _, m := range regexp.MustCompile(`e\.data\.(gkill_[a-z_]+)`).FindAllStringSubmatch(out, -1) {
		keys[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`postMessage\(\{ (gkill_[a-z_]+)`).FindAllStringSubmatch(out, -1) {
		keys[m[1]] = true
	}
	want := map[string]bool{"gkill_theme": true, "gkill_plugin_config": true, "gkill_plugin_config_result": true}
	if !maps.Equal(keys, want) {
		t.Errorf("postMessage のキーが増減している: %v", keys)
	}
}

func TestRenderEscapesEveryField(t *testing.T) {
	out := Render(Page{
		Title:    `<title>`,
		Intro:    `<intro>`,
		Status:   &Status{Headline: `<xhead>`, Lines: []string{`<line>`}, Steps: []string{`<step>`}},
		Progress: Progress{State: "ingesting", Done: 1, Total: 2, Unit: `<unit>`, Note: `<note>`},
		Stats:    []Row{{Label: `<xlabel>`, Value: `<value>`}, {Label: "k", Value: `<path>`, Code: true}},
		Notices:  []Notice{{Title: `<ntitle>`, Text: `<ntext>`, Items: []Item{{Label: `<ilabel>`, Code: `<icode>`, Text: `<itext>`, Tag: `<itag>`}}}},
		Patterns: []string{`<pattern>`},
		Expanded: &Expanded{Dirs: []string{`<dir>`}, Files: []string{`<file>`}},
		Sections: []Section{
			{Title: `<stitle>`, Items: []Item{{Code: `<sitem>`}}, Note: `<snote>`},
			{Title: `<ttitle>`, Table: &Table{Header: []string{`<xth>`}, Rows: [][]string{{`<xtd>`}}}},
			{Title: `<etitle>`, Empty: `<empty>`},
		},
		Form: &Form{Title: `<ftitle>`, Intro: `<fintro>`, Fields: []Field{
			{Key: "source_dirs", Label: `<flabel>`, Value: `<fvalue>`},
			{Key: "tz", Kind: FieldText, Label: `<tlabel>`, Value: `"><script>`},
			{Key: "flag", Kind: FieldCheckbox, Label: `<clabel>`},
		}},
		ConfigFile: &ConfigFile{Path: `C:/<p>/config.json`, Sample: `{"a": "<s>"}`, Notes: []string{`<cnote>`}},
	})
	for _, forbidden := range []string{
		`<title>`, `<intro>`, `<xhead>`, `<line>`, `<step>`, `<unit>`, `<note>`, `<xlabel>`, `<value>`, `<path>`,
		`<ntitle>`, `<ntext>`, `<ilabel>`, `<icode>`, `<itext>`, `<itag>`, `<pattern>`, `<dir>`, `<file>`,
		`<stitle>`, `<sitem>`, `<snote>`, `<ttitle>`, `<xth>`, `<xtd>`, `<etitle>`, `<empty>`,
		`<ftitle>`, `<fintro>`, `<flabel>`, `<fvalue>`, `"><script>`, `<tlabel>`, `<clabel>`,
		`C:/<p>/`, `"<s>"`, `<cnote>`,
	} {
		if strings.Contains(out, forbidden) {
			t.Errorf("エスケープされていない: %q", forbidden)
		}
	}
	for _, want := range []string{
		`<h2>&lt;title&gt;</h2>`, `<code>&lt;path&gt;</code>`, `<textarea id="gkill_source_dirs" rows="4" spellcheck="false">&lt;fvalue&gt;</textarea>`,
		`<input type="text" id="gkill_tz" value="&#34;&gt;&lt;script&gt;">`, `<code>C:/&lt;p&gt;/config.json</code>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("エスケープ後の文字列が無い: %q", want)
		}
	}
}

func TestRenderStatusLoadedOrNot(t *testing.T) {
	loaded := Render(Page{Status: &Status{Loaded: true, Headline: "**3 件**を読み込んでいます", Lines: []string{"更新するには置き換える"}}})
	if !strings.Contains(loaded, `<div class="ok"><p>✓ <strong>3 件</strong>を読み込んでいます</p><p>更新するには置き換える</p></div>`) {
		t.Errorf("読み込み済みの帯の形が違う:\n%s", loaded)
	}
	pending := Render(Page{Status: &Status{Headline: "まだ読み込まれていません。", Lines: []string{"手順:"}, Steps: []string{"**Settings** を開く", "ZIP を `dir` に置く"}}})
	if !strings.Contains(pending, `<div class="warn"><p><strong>まだ読み込まれていません。</strong></p><p>手順:</p><ol><li><strong>Settings</strong> を開く</li><li>ZIP を <code>dir</code> に置く</li></ol></div>`) {
		t.Errorf("未読み込みの帯の形が違う:\n%s", pending)
	}
	if strings.Contains(Render(Page{}), `class="ok"`) || strings.Contains(Render(Page{}), `class="warn"`) {
		t.Error("Status が nil なのに帯を出している")
	}
}

func TestRenderProgressStates(t *testing.T) {
	for _, state := range []string{"", "idle", "error"} {
		if out := Render(Page{Progress: Progress{State: state, Done: 1, Total: 2}}); strings.Contains(out, `class="progress"`) || strings.Contains(out, "取り込み中") {
			t.Errorf("state=%q で進捗を出している", state)
		}
	}
	scanning := Render(Page{Progress: Progress{State: "scanning"}})
	if !strings.Contains(scanning, `<div class="ok"><p>データソースを走査しています…</p></div>`) {
		t.Errorf("走査中の形が違う:\n%s", scanning)
	}
	ingesting := Render(Page{Progress: Progress{State: "ingesting", Done: 1234, Total: 5678, Unit: "ファイル", Note: "初回は時間がかかります"}})
	for _, want := range []string{
		`<p>取り込み中 1,234 / 5,678 ファイル (21%)</p>`,
		`<div class="progress"><div style="width:21%"></div></div>`,
		`<div class="hint">初回は時間がかかります</div>`,
	} {
		if !strings.Contains(ingesting, want) {
			t.Errorf("%q が無い:\n%s", want, ingesting)
		}
	}
	if out := Render(Page{Progress: Progress{State: "folding", Note: "畳み直し待ち 3 グループ"}}); !strings.Contains(out, `<p>集計中</p><div class="hint">畳み直し待ち 3 グループ</div>`) {
		t.Errorf("集計中の形が違う:\n%s", out)
	}
	if out := Render(Page{Progress: Progress{State: "something_else"}}); !strings.Contains(out, `<p>処理中…</p>`) {
		t.Errorf("未知の状態の形が違う:\n%s", out)
	}
	// Done が Total を超えても 100% で止める
	if out := Render(Page{Progress: Progress{State: "ingesting", Done: 30, Total: 10}}); !strings.Contains(out, `(100%)`) {
		t.Errorf("100%% で止めていない:\n%s", out)
	}
}

func TestRenderStatsRows(t *testing.T) {
	out := Render(Page{Stats: []Row{{Label: "会話数", Value: "1,234"}, {Label: "最終スキャン", Value: ""}, {Label: "キャッシュDB", Value: "C:/x/cache.db", Code: true}}})
	if !strings.Contains(out, `<table><tr><td class="k">会話数</td><td>1,234</td></tr><tr><td class="k">最終スキャン</td><td>—</td></tr><tr><td class="k">キャッシュDB</td><td><code>C:/x/cache.db</code></td></tr></table>`) {
		t.Errorf("統計表の形が違う:\n%s", out)
	}
	if strings.Contains(Render(Page{}), `<table>`) {
		t.Error("統計が無いのに表を出している")
	}
}

func TestRenderNoticesSkipEmpty(t *testing.T) {
	out := Render(Page{Notices: []Notice{
		{},
		MissingPatterns(nil),
		ErrorNotice("スキャン時のエラー:", ""),
		MissingPatterns([]string{"D:/nowhere/*.zip"}),
		ErrorNotice("スキャン時のエラー:", "open failed"),
		{Title: "大きすぎて読み飛ばした行が **3 行**あります。", Text: "ログの形式が変わった可能性があります。"},
	}})
	if got := strings.Count(out, `<div class="warn">`); got != 3 {
		t.Errorf("空の警告を飛ばしていない。警告の数 = %d:\n%s", got, out)
	}
	for _, want := range []string{
		`<div class="warn"><p><strong>次の指定は何にもマッチしませんでした:</strong></p><ul><li><code>D:/nowhere/*.zip</code></li></ul></div>`,
		`<div class="warn"><p><strong>スキャン時のエラー:</strong></p><ul><li><code>open failed</code></li></ul></div>`,
		`<div class="warn"><p><strong>大きすぎて読み飛ばした行が <strong>3 行</strong>あります。</strong></p><p>ログの形式が変わった可能性があります。</p></div>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q が無い:\n%s", want, out)
		}
	}
}

func TestRenderPatternsTagOnlyForGlob(t *testing.T) {
	out := Render(Page{Patterns: []string{"D:/repos/*.zip", "D:/one.zip"}})
	if !strings.Contains(out, `<h3>設定されている指定</h3><ul><li><code>D:/repos/*.zip</code> <span class="tag">パターン</span></li><li><code>D:/one.zip</code></li></ul>`) {
		t.Errorf("指定一覧の形が違う:\n%s", out)
	}
	if strings.Contains(Render(Page{}), "設定されている指定") {
		t.Error("指定が無いのに節を出している")
	}
}

func TestRenderExpandedCapsAtDefault(t *testing.T) {
	e := &Expanded{Dirs: []string{"D:/a"}}
	for i := range 25 {
		e.Files = append(e.Files, "D:/f"+string(rune('a'+i)))
	}
	out := Render(Page{Expanded: e})
	for _, want := range []string{
		`<h3>展開結果 (フォルダ 1 / ファイル 25)</h3>`,
		`<li><code>D:/a</code> (再帰的に走査)</li>`,
		`<li>ほか 6 件</li>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q が無い:\n%s", want, out)
		}
	}
	if strings.Contains(out, "D:/fu") {
		t.Error("上限を超えたファイルを出している")
	}
	if strings.Contains(Render(Page{}), "展開結果") {
		t.Error("Expanded が nil なのに節を出している")
	}
}

func TestRenderSections(t *testing.T) {
	out := Render(Page{Sections: []Section{
		{Title: "取り込んだリポジトリ (2)", Items: []Item{{Label: "a", Code: "D:/a.zip!/a/.git", Text: "2 コミット"}}},
		{Title: "書き出し", Table: &Table{Header: []string{"フォルダ", "ZIP数"}, Rows: [][]string{{"D:/t1", "3"}, {"D:/t2", "1"}}}, Max: 1, Note: "分割ZIPは1つに合算"},
		{Title: "指標一覧", Collapsed: true, Table: &Table{Rows: [][]string{{"steps"}}}, Max: -1},
		{Title: "書き出し（空）", Empty: "ZIPがまだ見つかっていません。"},
		{Title: "出さない節"},
	}})
	for _, want := range []string{
		`<h3>取り込んだリポジトリ (2)</h3><ul><li><strong>a</strong> <code>D:/a.zip!/a/.git</code> 2 コミット</li></ul>`,
		`<h3>書き出し</h3><table><tr><th>フォルダ</th><th>ZIP数</th></tr><tr><td>D:/t1</td><td>3</td></tr></table><p class="hint">ほか 1 件</p><p class="hint">分割ZIPは1つに合算</p>`,
		`<details><summary>指標一覧</summary><table><tr><td>steps</td></tr></table></details>`,
		`<h3>書き出し（空）</h3><p class="hint">ZIPがまだ見つかっていません。</p>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q が無い:\n%s", want, out)
		}
	}
	if strings.Contains(out, "出さない節") {
		t.Error("0件で Empty も無い節を出している")
	}
}

func TestRenderConfigFileCollapsed(t *testing.T) {
	out := Render(Page{ConfigFile: &ConfigFile{Path: "C:/plugins/x/config.json", Sample: "{\n  \"source_dirs\": []\n}", Notes: []string{"", "`source_dirs` は**配列**"}}})
	for _, want := range []string{
		`<details><summary>config.json を直接編集する</summary>`,
		`<code>manifest.json</code>`,
		`<p>編集するファイル: <code>C:/plugins/x/config.json</code></p>`,
		"<pre>{\n  &#34;source_dirs&#34;: []\n}</pre>",
		`<div class="hint"><code>source_dirs</code> は<strong>配列</strong></div></details>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q が無い:\n%s", want, out)
		}
	}
	if strings.Contains(Render(Page{}), "config.json を直接編集する") {
		t.Error("ConfigFile が nil なのに案内を出している")
	}
}

func TestRenderOrder(t *testing.T) {
	out := Render(Page{
		Title:      "題",
		Intro:      "補足",
		Status:     &Status{Loaded: true, Headline: "読み込み済み"},
		Progress:   Progress{State: "scanning"},
		Stats:      []Row{{Label: "件数", Value: "1"}},
		Notices:    []Notice{{Title: "警告"}},
		Patterns:   []string{"D:/p"},
		Expanded:   &Expanded{Dirs: []string{"D:/d"}},
		Sections:   []Section{{Title: "追加の節", Items: []Item{{Code: "x"}}}},
		Form:       &Form{Fields: []Field{{Key: "source_dirs"}}},
		ConfigFile: &ConfigFile{Path: "C:/c"},
	})
	order := []string{
		`<h2>題</h2>`, `<p class="hint">補足</p>`, `読み込み済み`, `データソースを走査しています…`, `<td class="k">件数</td>`,
		`<strong>警告</strong>`, `<h3>設定されている指定</h3>`, `<h3>展開結果`, `<h3>追加の節</h3>`,
		`<h3>設定</h3>`, `id="gkill_save"`, `<summary>config.json を直接編集する</summary>`, `</body></html>`,
	}
	last := -1
	for _, want := range order {
		at := strings.Index(out, want)
		if at < 0 {
			t.Fatalf("%q が無い:\n%s", want, out)
		}
		if at < last {
			t.Errorf("%q の位置が前の要素より前にある", want)
		}
		last = at
	}
}
