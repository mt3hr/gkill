package configview

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// FieldKind は編集フォームの欄の種類。ゼロ値は複数行（source_dirs が常にこれなので）。
type FieldKind int

const (
	// FieldTextarea は複数行のテキスト（1行に1つ書く指定・一覧）。
	FieldTextarea FieldKind = iota
	// FieldText は1行のテキスト（タイムゾーン・数値）。
	FieldText
	// FieldCheckbox は真偽値。保存時は '1'（on）か ''（off）で送る（PostConfig は value == "1" で読む）。
	FieldCheckbox
)

// Form は編集フォーム。保存ボタンと、親（設定ダイアログ）へ保存を依頼するスクリプトを付けて描く。
//
// iframe は allow-same-origin なしで動くため自力では API を叩けない。保存は gkill 本体の設定ダイアログが
// postMessage で肩代わりする（use-plugin-config-dialog.ts）。
//
//	iframe → 親 : { gkill_plugin_config: { <Key>: <値>, ... } }
//	親 → iframe : { gkill_plugin_config_result: { ok, error } }
//
// 値は全部文字列で送る（親も文字列しか受け付けない）。欄の要素 id は gkill_<Key>。
type Form struct {
	// Title は見出し。空なら「設定」。
	Title string
	// Intro は見出しの下の文（Prose 記法）。
	Intro string
	// Fields は欄。上から順に描く。
	Fields []Field
}

// Field は編集フォームの欄1つ。
type Field struct {
	// Key は config.json のキー（snake_case）。postMessage のキーと要素 id（gkill_<Key>）になる。
	// 英小文字で始まり英小文字・数字・_ だけ（validFieldKey）。外れた欄は描かずに警告を出す。
	Key string
	// Label は欄の上の説明（Prose 記法）。チェックボックスでは右側の文。
	Label string
	// Kind は欄の種類。
	Kind FieldKind
	// Value は現在値（Textarea / Text）。
	Value string
	// Checked は現在値（Checkbox）。
	Checked bool
	// Rows は Textarea の行数。0 なら 4。
	Rows int
}

// validFieldKey は Key の形。JS の識別子・HTML の id・JSON のキーのどれとしても安全な範囲。
var validFieldKey = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// jsReservedWords は var 名にできない語。Key の camelCase がこれに当たると "Field" を後ろに付ける。
// 当たったまま出すとスクリプト全体が構文エラーで死に、保存ボタンが何もしなくなる（エラーも出ない）。
var jsReservedWords = map[string]bool{
	"break": true, "case": true, "catch": true, "class": true, "const": true, "continue": true,
	"debugger": true, "default": true, "delete": true, "do": true, "else": true, "enum": true,
	"export": true, "extends": true, "false": true, "finally": true, "for": true, "function": true,
	"if": true, "import": true, "in": true, "instanceof": true, "new": true, "null": true,
	"return": true, "super": true, "switch": true, "this": true, "throw": true, "true": true,
	"try": true, "typeof": true, "var": true, "void": true, "while": true, "with": true, "yield": true,
	"let": true, "static": true, "await": true,
	// スクリプト内で使っている名前
	"btn": true, "out": true, "e": true, "r": true, "parent": true, "window": true, "document": true,
}

// jsVar は Key から保存スクリプトの var 名を作る（secondary_data_sources → secondaryDataSources）。
func jsVar(key string) string {
	var sb strings.Builder
	upper := false
	for _, r := range key {
		if r == '_' {
			upper = true
			continue
		}
		if upper {
			sb.WriteString(strings.ToUpper(string(r)))
			upper = false
		} else {
			sb.WriteRune(r)
		}
	}
	name := sb.String()
	if name == "" || jsReservedWords[name] {
		name += "Field"
	}
	return name
}

// fieldID は欄の要素 id。
func fieldID(key string) string {
	return "gkill_" + key
}

// renderForm は編集フォームと保存スクリプトを描く。
func renderForm(sb *strings.Builder, f *Form) {
	if f == nil || len(f.Fields) == 0 {
		return
	}
	var fields []Field
	for _, fd := range f.Fields {
		if !validFieldKey.MatchString(fd.Key) {
			// 黙って落とすと「設定したつもりの値が保存されない」になるので、画面に出す
			sb.WriteString(`<div class="warn"><p>フォームのキーが不正なので欄を出せません: <code>` +
				html.EscapeString(fd.Key) + `</code></p></div>`)
			continue
		}
		fields = append(fields, fd)
	}
	if len(fields) == 0 {
		return
	}

	title := f.Title
	if title == "" {
		title = "設定"
	}
	sb.WriteString(`<h3>` + html.EscapeString(title) + `</h3>`)
	if f.Intro != "" {
		sb.WriteString(`<p>` + Prose(f.Intro) + `</p>`)
	}
	for _, fd := range fields {
		id := fieldID(fd.Key)
		switch fd.Kind {
		case FieldCheckbox:
			checked := ""
			if fd.Checked {
				checked = " checked"
			}
			fmt.Fprintf(sb, `<label><input type="checkbox" id="%s"%s> %s</label>`, id, checked, Prose(fd.Label))
		case FieldText:
			if fd.Label != "" {
				fmt.Fprintf(sb, `<label for="%s">%s</label>`, id, Prose(fd.Label))
			}
			fmt.Fprintf(sb, `<input type="text" id="%s" value="%s">`, id, html.EscapeString(fd.Value))
		default:
			rows := fd.Rows
			if rows <= 0 {
				rows = 4
			}
			if fd.Label != "" {
				fmt.Fprintf(sb, `<label for="%s">%s</label>`, id, Prose(fd.Label))
			}
			fmt.Fprintf(sb, `<textarea id="%s" rows="%d" spellcheck="false">%s</textarea>`, id, rows, html.EscapeString(fd.Value))
		}
	}
	sb.WriteString(`<p><button type="button" id="gkill_save">保存</button><span id="gkill_save_result"></span></p>`)
	sb.WriteString(saveScript(fields))
}

// saveScript は設定ダイアログ（親）へ保存を依頼するスクリプト。欄の値を Key ごとに集めて1つの postMessage に載せる。
func saveScript(fields []Field) string {
	var sb strings.Builder
	sb.WriteString("<script>\n(function () {\n")
	for _, fd := range fields {
		fmt.Fprintf(&sb, "  var %s = document.getElementById('%s');\n", jsVar(fd.Key), fieldID(fd.Key))
	}
	sb.WriteString("  var btn = document.getElementById('gkill_save');\n")
	sb.WriteString("  var out = document.getElementById('gkill_save_result');\n")
	sb.WriteString("  if (!btn || !out) { return; }\n")
	sb.WriteString("  btn.addEventListener('click', function () {\n")
	sb.WriteString("    btn.disabled = true;\n")
	sb.WriteString("    out.textContent = '保存中…';\n")
	sb.WriteString("    parent.postMessage({ gkill_plugin_config: {\n")
	for i, fd := range fields {
		v := jsVar(fd.Key)
		if fd.Kind == FieldCheckbox {
			fmt.Fprintf(&sb, "      %s: (%s && %s.checked) ? '1' : ''", fd.Key, v, v)
		} else {
			fmt.Fprintf(&sb, "      %s: %s ? %s.value : ''", fd.Key, v, v)
		}
		if i < len(fields)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("    } }, '*');\n")
	sb.WriteString("  });\n")
	sb.WriteString("  window.addEventListener('message', function (e) {\n")
	sb.WriteString("    var r = e.data && e.data.gkill_plugin_config_result;\n")
	sb.WriteString("    if (!r) { return; }\n")
	sb.WriteString("    btn.disabled = false;\n")
	sb.WriteString("    out.textContent = r.ok ? '保存しました' : ('保存に失敗しました: ' + (r.error || ''));\n")
	sb.WriteString("  });\n")
	sb.WriteString("})();\n</script>")
	return sb.String()
}
