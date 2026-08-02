package message

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"testing"
)

// エラーコード / メッセージコードは、クライアントが error_code で分岐したり
// ログを追跡したりするための識別子。重複すると別々の障害が同じコードになって
// 区別できなくなる。
//
// 以前はここに定数名を手書きした map / slice があり、それを対象に
// 「空でない」「重複しない」「形式が正しい」を確認していた。しかし
//   - 「空でない」は const 宣言時点でコンパイラが保証している
//   - 手書きゆえ、新しい定数を追加して書き忘れると重複チェックを素通りする
//     （実際 error_codes.go の400定数に対し、重複を見ていたのは29個だけだった）
//
// ため、ソースを go/parser で読んで全定数を対象にする形に変えている。
// 新しいコードを足しても、テスト側を触らずに検査対象へ入る。
//
// この網羅チェックを入れた時点で message_codes.go に重複が2件あった
// （MSG000027: GetKmemo/GetURLog、MSG000045: UpdateTagStruct/UpdateRepStruct）。
// いずれもコピペ時に採番を振り直し忘れたもので、GetURLogSuccessMessage を
// MSG000084、UpdateRepStructSuccessMessage を MSG000085 に振り直して解消済み。

// parseConstStrings は指定ファイルのトップレベル const 宣言から
// 「定数名 → 文字列リテラル値」を取り出す。
func parseConstStrings(t *testing.T, filename string) map[string]string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}

	consts := map[string]string{}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				lit, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				// リテラルの前後のダブルクォートを外す
				consts[name.Name] = lit.Value[1 : len(lit.Value)-1]
			}
		}
	}

	if len(consts) == 0 {
		t.Fatalf("%s から定数を1つも読み取れなかった（定義の書き方が変わった可能性がある）", filename)
	}
	return consts
}

// assertUniqueAndFormatted は全定数が一意で、指定の形式に合っていることを確認する。
func assertUniqueAndFormatted(t *testing.T, consts map[string]string, pattern *regexp.Regexp) {
	t.Helper()

	seenBy := map[string]string{}
	for name, value := range consts {
		if value == "" {
			t.Errorf("%s が空文字", name)
			continue
		}
		if !pattern.MatchString(value) {
			t.Errorf("%s = %q が形式 %s に合っていない", name, value, pattern)
		}
		if first, dup := seenBy[value]; dup {
			t.Errorf("コード %q が重複している: %s と %s", value, first, name)
			continue
		}
		seenBy[value] = name
	}
}

// TestErrorCodes_UniqueAndFormatted は error_codes.go の全定数を対象に
// ERR + 6桁 の形式と一意性を確認する。
func TestErrorCodes_UniqueAndFormatted(t *testing.T) {
	consts := parseConstStrings(t, "error_codes.go")
	t.Logf("検査対象のエラーコード: %d 件", len(consts))
	assertUniqueAndFormatted(t, consts, regexp.MustCompile(`^ERR\d{6}$`))
}

// TestMessageCodes_UniqueAndFormatted は message_codes.go の全定数を対象に
// MSG + 6桁 の形式と一意性を確認する。
func TestMessageCodes_UniqueAndFormatted(t *testing.T) {
	consts := parseConstStrings(t, "message_codes.go")
	t.Logf("検査対象のメッセージコード: %d 件", len(consts))
	assertUniqueAndFormatted(t, consts, regexp.MustCompile(`^MSG\d{6}$`))
}

// TestErrorAndMessageCodes_DoNotCollide はエラーコードとメッセージコードで
// 同じ値が使われていないことを確認する。接頭辞が違うので本来ぶつからないが、
// 片方に ERR/MSG を取り違えて書いた場合をここで拾う。
func TestErrorAndMessageCodes_DoNotCollide(t *testing.T) {
	errorCodes := parseConstStrings(t, "error_codes.go")
	messageCodes := parseConstStrings(t, "message_codes.go")

	errorValues := map[string]string{}
	for name, value := range errorCodes {
		errorValues[value] = name
	}
	for name, value := range messageCodes {
		if errName, collides := errorValues[value]; collides {
			t.Errorf("コード %q がエラー(%s)とメッセージ(%s)の両方で使われている", value, errName, name)
		}
	}
}

// TestGkillError_JSONRoundTrip は、クライアントが参照するフィールド名を固定する。
func TestGkillError_JSONRoundTrip(t *testing.T) {
	original := GkillError{
		ErrorCode:    AccountNotFoundError,
		ErrorMessage: "Account was not found",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}
	for _, field := range []string{"error_code", "error_message"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("JSON にフィールド %q が無い: %s", field, data)
		}
	}

	var restored GkillError
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if restored != original {
		t.Errorf("restored = %+v, want %+v", restored, original)
	}
}

// TestGkillMessage_JSONRoundTrip は、クライアントが参照するフィールド名を固定する。
func TestGkillMessage_JSONRoundTrip(t *testing.T) {
	original := GkillMessage{
		MessageCode: LoginSuccessMessage,
		Message:     "Login successful",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}
	for _, field := range []string{"message_code", "message"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("JSON にフィールド %q が無い: %s", field, data)
		}
	}

	var restored GkillMessage
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if restored != original {
		t.Errorf("restored = %+v, want %+v", restored, original)
	}
}
