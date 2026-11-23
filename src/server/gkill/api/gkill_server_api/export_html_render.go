package gkill_server_api

// 「HTML で出力」の組み立て。ガワ（embed/export/template.html）へ焼き込みを差し込む。
//
// ガワは npm run prepare_install が作って embed へ写す。
// **裸の go build では embed に無い。** そのときはこの API だけが 500 を返し、
// ほかの機能は動く（init() で panic しない）。

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"unicode"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// exportedPayloadSchemaVersion は焼き込みの世代。
// クライアント側の exported_payload_schema_version（exported-kyou-payload.ts）と対。
const exportedPayloadSchemaVersion = 1

const (
	// exportTemplatePath はガワの置き場。embed/html/ へ置くと serve.go の FileServer から
	// /export.html として配信されてしまうので、別のディレクトリにしている
	exportTemplatePath = "embed/export/template.html"
	// exportTitlePlaceholder と exportDataMarker は export.html が持つ差し込み先。
	// ガワのビルド（src/tools/build_export_template.mjs）と対
	exportTitlePlaceholder = "{{GKILL_EXPORT_TITLE}}"
	exportDataMarker       = `<script type="application/json" id="gkill-export-data">{}</script>`
)

// exportedColumn は出力する列1本。
//
// 記録の本体は kyou_ids の参照で持つ（列をまたいで同じ記録が出るのは普通で、
// 列ごとに本体を持つと素直に膨らむ）。
// Query は**クライアントが送ってきた JSON そのまま**（req_res.ExportRykvHTMLColumn.Raw）。
// find.FindQuery へ型に落として再 marshal すると query_id などが消え、
// 出力先が列を引けなくなる（2列以上で全列が先頭列に一致する）。
type exportedColumn struct {
	Query   json.RawMessage `json:"query"`
	KyouIDs []string        `json:"kyou_ids"`
}

// exportedMediaEntry は焼き込んだメディア1件。
// Src は単一 HTML なら data: URI、同名フォルダなら相対パス。
type exportedMediaEntry struct {
	Src  string `json:"src"`
	Kind string `json:"kind"`
}

// exportedIDFTextContent は IDF のテキスト / Markdown の本文。
//
// **バイト列のまま焼く。** 文字コードの判定はクライアントの detect_and_decode_text が
// 唯一の判定者で、ここでデコードすると2実装になって必ずずれる（ADR-0416）。
type exportedIDFTextContent struct {
	BytesBase64 string `json:"bytes_base64"`
	Truncated   bool   `json:"truncated"`
}

// exportedKyouPayload は HTML へ焼き込む JSON。
// クライアント側の読み手は exported-kyou-payload.ts の build_exported_api。
type exportedKyouPayload struct {
	SchemaVersion int      `json:"schema_version"`
	Title         string   `json:"title"`
	LocaleName    string   `json:"locale_name"`
	WithDnote     bool     `json:"with_dnote"`
	WithMap       bool     `json:"with_map"`
	WithSidebar   bool     `json:"with_sidebar"`
	Warnings      []string `json:"warnings"`

	// ApplicationConfig は焼き込む設定。**google_map_api_key は必ず空にする**
	// （落とすと利用者のキーが共有先へ漏れる）
	ApplicationConfig *user_config.ApplicationConfig `json:"application_config"`

	Columns []exportedColumn `json:"columns"`

	Kyous         []reps.Kyou         `json:"kyous"`
	Kmemos        []reps.Kmemo        `json:"kmemos"`
	KCs           []reps.KC           `json:"kcs"`
	TimeIss       []reps.TimeIs       `json:"timeiss"`
	Mis           []reps.Mi           `json:"mis"`
	Nlogs         []reps.Nlog         `json:"nlogs"`
	Lantanas      []reps.Lantana      `json:"lantanas"`
	URLogs        []reps.URLog        `json:"urlogs"`
	IDFKyous      []reps.IDFKyou      `json:"idf_kyous"`
	ReKyous       []reps.ReKyou       `json:"rekyous"`
	MiReKyous     []reps.MiReKyou     `json:"mirekyous"`
	GitCommitLogs []reps.GitCommitLog `json:"git_commit_logs"`
	GPSLogs       []reps.GPSLog       `json:"gps_logs"`

	AttachedTags          []reps.Tag          `json:"attached_tags"`
	AttachedTexts         []reps.Text         `json:"attached_texts"`
	AttachedNotifications []reps.Notification `json:"attached_notifications"`
	AttachedTimeIss       []reps.TimeIs       `json:"attached_timeiss"`
	AttachedTimeIsKyous   []reps.Kyou         `json:"attached_timeis_kyous"`

	PluginHTML      map[string]string                 `json:"plugin_html"`
	Media           map[string]exportedMediaEntry     `json:"media"`
	IDFTextContents map[string]exportedIDFTextContent `json:"idf_text_contents"`
}

// readExportTemplate はガワを読む。embed に無ければエラー（裸の go build）。
func readExportTemplate() ([]byte, error) {
	template, err := api.EmbedFS.ReadFile(exportTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("error at read export template %s: %w", exportTemplatePath, err)
	}
	return template, nil
}

// renderExportHTML はガワへ焼き込みを差し込んだ HTML を返す。
//
// localeMessages は ja 以外で出力するときの i18n（nil なら差し込まない。ガワは ja を内蔵している）。
func renderExportHTML(template []byte, payload *exportedKyouPayload, localeMessages map[string]string) (string, error) {
	// **json.Marshal の既定の HTML エスケープに依存している。**
	// 記録の本文に </script> が1つあるだけで、ブラウザがそこから先を HTML として
	// 解釈して画面が丸ごと壊れる。encoding/json は既定で < > & を < > & に
	// するので割れない。SetEscapeHTML(false) に書き換えないこと
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error at marshal export payload: %w", err)
	}

	rendered := string(template)
	if !strings.Contains(rendered, exportDataMarker) {
		return "", fmt.Errorf("error at find export data marker in template %s", exportTemplatePath)
	}

	elements := `<script type="application/json" id="gkill-export-data">` + string(encoded) + `</script>`
	if len(localeMessages) != 0 {
		// i18n も同じ理由で json.Marshal を通す（生の JSON をそのまま挟まない）
		encodedLocale, err := json.Marshal(localeMessages)
		if err != nil {
			return "", fmt.Errorf("error at marshal export locale messages: %w", err)
		}
		elements += `<script type="application/json" id="gkill-export-i18n">` + string(encodedLocale) + `</script>`
	}

	rendered = strings.Replace(rendered, exportDataMarker, elements, 1)
	rendered = strings.ReplaceAll(rendered, exportTitlePlaceholder, html.EscapeString(payload.Title))
	return rendered, nil
}

// exportFileNameBase は出力ファイルの名前を表題から作る。
//
// 制御文字・パス区切り・Windows で使えない文字を落とす。
// 同名フォルダの名前にもなるので、ここが通らない文字を残すと保存できない。
func exportFileNameBase(title string) string {
	const fallback = "gkill_export"
	var builder strings.Builder
	for _, r := range title {
		switch {
		case unicode.IsControl(r):
			continue
		case strings.ContainsRune(`\/:*?"<>|`, r):
			continue
		case r == '.' && builder.Len() == 0:
			// 先頭のドットは隠しファイルになるので落とす
			continue
		default:
			builder.WriteRune(r)
		}
	}
	name := strings.TrimSpace(builder.String())
	name = strings.TrimRight(name, ". ")
	if name == "" {
		return fallback
	}
	// 長すぎる名前は保存に失敗する（Windows の MAX_PATH）。字数で切る
	runes := []rune(name)
	if len(runes) > 60 {
		name = strings.TrimSpace(string(runes[:60]))
	}
	if name == "" {
		return fallback
	}
	return name
}
