package configview

import (
	"strings"
	"testing"
)

func TestRenderFormFieldsAndSaveScript(t *testing.T) {
	out := Render(Page{Form: &Form{Intro: "1行に1つ。", Fields: []Field{
		{Key: "source_dirs", Label: "取り込み元", Value: "D:/a\nD:/b", Rows: 6},
		{Key: "timezone", Kind: FieldText, Label: "タイムゾーン", Value: "Asia/Tokyo"},
		{Key: "secondary_data_sources", Value: "Phone"},
		{Key: "include_fitbit_gps", Kind: FieldCheckbox, Label: "トラックを含める", Checked: true},
		{Key: "visit_points", Kind: FieldCheckbox, Label: "端点も出す"},
	}}})
	for _, want := range []string{
		`<h3>設定</h3><p>1行に1つ。</p>`,
		`<label for="gkill_source_dirs">取り込み元</label><textarea id="gkill_source_dirs" rows="6" spellcheck="false">D:/a` + "\n" + `D:/b</textarea>`,
		`<label for="gkill_timezone">タイムゾーン</label><input type="text" id="gkill_timezone" value="Asia/Tokyo">`,
		`<textarea id="gkill_secondary_data_sources" rows="4" spellcheck="false">Phone</textarea>`,
		`<label><input type="checkbox" id="gkill_include_fitbit_gps" checked> トラックを含める</label>`,
		`<label><input type="checkbox" id="gkill_visit_points"> 端点も出す</label>`,
		`<p><button type="button" id="gkill_save">保存</button><span id="gkill_save_result"></span></p>`,
		// 保存スクリプト。var 名は Key の camelCase（fitbit の html_test.go がこの形を見る）
		`var secondaryDataSources = document.getElementById('gkill_secondary_data_sources');`,
		`source_dirs: sourceDirs ? sourceDirs.value : ''`,
		`timezone: timezone ? timezone.value : ''`,
		`secondary_data_sources: secondaryDataSources ? secondaryDataSources.value : ''`,
		`include_fitbit_gps: (includeFitbitGps && includeFitbitGps.checked) ? '1' : ''`,
		`visit_points: (visitPoints && visitPoints.checked) ? '1' : ''`,
		`parent.postMessage({ gkill_plugin_config: {`,
		`e.data.gkill_plugin_config_result`,
		`'保存中…'`, `'保存しました'`, `'保存に失敗しました: '`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q が無い:\n%s", want, out)
		}
	}
	// 最後の欄の後ろにカンマを置かない（古い WebView で構文エラー）
	if strings.Contains(out, "'1' : '',\n    } }") || strings.Contains(out, ": '',\n    } }") {
		t.Error("オブジェクトの末尾にカンマが残っている")
	}
}

func TestRenderFormNilOrEmptyIsSkipped(t *testing.T) {
	for _, out := range []string{Render(Page{}), Render(Page{Form: &Form{}})} {
		if strings.Contains(out, `id="gkill_save"`) || strings.Contains(out, "<script>\n(function") {
			t.Error("欄が無いのにフォームや保存スクリプトを出している")
		}
	}
}

func TestRenderFormInvalidKeyIsReportedNotDropped(t *testing.T) {
	out := Render(Page{Form: &Form{Fields: []Field{{Key: "Bad-Key'", Value: "x"}, {Key: "source_dirs"}}}})
	if !strings.Contains(out, `<div class="warn"><p>フォームのキーが不正なので欄を出せません: <code>Bad-Key&#39;</code></p></div>`) {
		t.Errorf("不正なキーを画面に出していない:\n%s", out)
	}
	if strings.Contains(out, "Bad-Key'") || strings.Contains(out, "gkill_Bad") {
		t.Error("不正なキーが HTML か スクリプトに漏れている")
	}
	if !strings.Contains(out, `id="gkill_source_dirs"`) {
		t.Error("正しい欄まで落としている")
	}
	// 全部不正なら保存ボタンも出さない（押しても何も送れない）
	if out := Render(Page{Form: &Form{Fields: []Field{{Key: "BAD"}}}}); strings.Contains(out, `id="gkill_save"`) {
		t.Error("出せる欄が無いのに保存ボタンを出している")
	}
}

func TestJSVar(t *testing.T) {
	cases := map[string]string{
		"source_dirs":            "sourceDirs",
		"secondary_data_sources": "secondaryDataSources",
		"timezone":               "timezone",
		"scan_workers":           "scanWorkers",
		"include_fitbit_gps":     "includeFitbitGps",
		// 予約語とスクリプト内の名前は衝突させない（衝突するとスクリプト全体が構文エラーで死ぬ）
		"default": "defaultField",
		"class":   "classField",
		"btn":     "btnField",
		"out":     "outField",
		"e":       "eField",
		"r":       "rField",
		"window":  "windowField",
	}
	for key, want := range cases {
		if got := jsVar(key); got != want {
			t.Errorf("jsVar(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestFieldKindZeroIsTextarea(t *testing.T) {
	if FieldTextarea != 0 {
		t.Error("FieldKind のゼロ値は Textarea（source_dirs が常にこれ）")
	}
}
