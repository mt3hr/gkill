package gkill_server_api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// 出力 HTML の組み立ては「静かに壊れる」ものを2つ抱えている。
//   - 記録の本文に </script> が1つあるだけで、ブラウザがそこから先を HTML として解釈し、
//     画面が丸ごと壊れる（焼き込みは <script type="application/json"> の中にある）
//   - 表題がそのまま <title> に入ると、タグを書いた表題で HTML が崩れる
//
// どちらも encoding/json の既定の HTML エスケープと html.EscapeString に依存している。
// 依存していることをここで固定する。

// テスト用のガワ。本物（embed/export/template.html）と同じ差し込み先だけを持つ。
const testExportTemplate = `<!doctype html><html><head><title>{{GKILL_EXPORT_TITLE}}</title></head>` +
	`<body><div id="app"></div>` +
	`<script type="application/json" id="gkill-export-data">{}</script>` +
	`</body></html>`

func minimalExportPayload(title string) *exportedKyouPayload {
	return &exportedKyouPayload{
		SchemaVersion:     exportedPayloadSchemaVersion,
		Title:             title,
		LocaleName:        "ja",
		Warnings:          []string{},
		ApplicationConfig: &user_config.ApplicationConfig{},
		Columns:           []exportedColumn{},
		Kyous:             []reps.Kyou{},
		Kmemos:            []reps.Kmemo{},
		PluginHTML:        map[string]string{},
		Media:             map[string]exportedMediaEntry{},
		IDFTextContents:   map[string]exportedIDFTextContent{},
	}
}

// TestRenderExportHTML_ScriptTagInContentDoesNotBreakDocument は、記録の本文に
// </script> が入っていても焼き込みの script が割れないことを確認する。
func TestRenderExportHTML_ScriptTagInContentDoesNotBreakDocument(t *testing.T) {
	payload := minimalExportPayload("表題")
	payload.Kmemos = []reps.Kmemo{{
		ID:      "k1",
		Content: `ここで割れる </script><script>alert(1)</script> はずだった`,
	}}

	rendered, err := renderExportHTML([]byte(testExportTemplate), payload, nil)
	if err != nil {
		t.Fatalf("renderExportHTML failed: %v", err)
	}

	// 焼き込みの script は1つだけ開いて1つだけ閉じる
	if strings.Count(rendered, "</script>") != 1 {
		t.Errorf("</script> が %d 個ある（焼き込みの script が割れている）", strings.Count(rendered, "</script>"))
	}
	if strings.Contains(rendered, "<script>alert(1)</script>") {
		t.Error("本文の script タグがそのまま HTML に出ている")
	}
	if !strings.Contains(rendered, `</script>`) {
		t.Error("encoding/json の HTML エスケープが効いていない（SetEscapeHTML(false) にされていないか）")
	}

	// エスケープされていても、読み出した側では元の文字列に戻る
	start := strings.Index(rendered, `id="gkill-export-data">`) + len(`id="gkill-export-data">`)
	end := strings.Index(rendered[start:], "</script>") + start
	var decoded map[string]any
	if err := json.Unmarshal([]byte(rendered[start:end]), &decoded); err != nil {
		t.Fatalf("焼き込みが JSON として読めない: %v", err)
	}
	kmemos, _ := decoded["kmemos"].([]any)
	if len(kmemos) != 1 {
		t.Fatalf("kmemos の件数 = %d, want 1", len(kmemos))
	}
	kmemo, _ := kmemos[0].(map[string]any)
	if !strings.Contains(kmemo["content"].(string), "</script>") {
		t.Error("読み出した本文が元に戻っていない")
	}
}

// TestRenderExportHTML_TitleIsEscaped は、表題がそのまま HTML に入らないことを確認する。
func TestRenderExportHTML_TitleIsEscaped(t *testing.T) {
	payload := minimalExportPayload(`</title><script>alert(1)</script>`)

	rendered, err := renderExportHTML([]byte(testExportTemplate), payload, nil)
	if err != nil {
		t.Fatalf("renderExportHTML failed: %v", err)
	}
	if strings.Contains(rendered, "<title></title><script>") {
		t.Error("表題がエスケープされずに HTML へ入っている")
	}
	if !strings.Contains(rendered, "&lt;/title&gt;") {
		t.Error("表題の < > がエスケープされていない")
	}
	if strings.Contains(rendered, exportTitlePlaceholder) {
		t.Error("表題の差し込み先が残っている")
	}
}

// TestRenderExportHTML_LocaleMessagesAreEmbeddedOnlyWhenGiven は、ja 以外で出力したときだけ
// i18n が差し込まれることを確認する（ガワは ja を内蔵している）。
func TestRenderExportHTML_LocaleMessagesAreEmbeddedOnlyWhenGiven(t *testing.T) {
	payload := minimalExportPayload("表題")

	withoutLocale, err := renderExportHTML([]byte(testExportTemplate), payload, nil)
	if err != nil {
		t.Fatalf("renderExportHTML failed: %v", err)
	}
	if strings.Contains(withoutLocale, "gkill-export-i18n") {
		t.Error("渡していないのに i18n が差し込まれている")
	}

	withLocale, err := renderExportHTML([]byte(testExportTemplate), payload,
		map[string]string{"SOME_KEY": `値に </script> が入っていても割れない`})
	if err != nil {
		t.Fatalf("renderExportHTML failed: %v", err)
	}
	if !strings.Contains(withLocale, `id="gkill-export-i18n"`) {
		t.Error("i18n が差し込まれていない")
	}
	if strings.Count(withLocale, "</script>") != 2 {
		t.Errorf("</script> が %d 個ある（i18n の script が割れている）", strings.Count(withLocale, "</script>"))
	}
}

// TestRenderExportHTML_BrokenTemplateIsRejected は、差し込み先の無いガワを弾くことを確認する。
// 黙って差し込まずに返すと、記録の入っていない HTML が配られる。
func TestRenderExportHTML_BrokenTemplateIsRejected(t *testing.T) {
	_, err := renderExportHTML([]byte(`<!doctype html><html></html>`), minimalExportPayload("表題"), nil)
	if err == nil {
		t.Error("差し込み先の無いガワを受け入れている")
	}
}

// TestExportFileNameBase は、保存できない名前を作らないことを確認する。
func TestExportFileNameBase(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		title string
		want  string
	}{
		{"そのまま通る", "2026年7月の記録", "2026年7月の記録"},
		{"パス区切りを落とす", "a/b\\c", "abc"},
		{"Windows で使えない文字を落とす", `a:b*c?d"e<f>g|h`, "abcdefgh"},
		{"空になったら既定の名前", "///", "gkill_export"},
		{"空の表題も既定の名前", "", "gkill_export"},
		{"先頭のドットを落とす（隠しファイルにしない）", ".gitignore", "gitignore"},
		{"末尾のドットと空白を落とす（Windows が開けない）", "記録. ", "記録"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := exportFileNameBase(testCase.title); got != testCase.want {
				t.Errorf("exportFileNameBase(%q) = %q, want %q", testCase.title, got, testCase.want)
			}
		})
	}
}
