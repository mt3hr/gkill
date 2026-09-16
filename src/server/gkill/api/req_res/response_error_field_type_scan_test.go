package req_res

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// 応答の Errors / Messages は名前付きスライス型（message.GkillErrors / message.GkillMessages）で持つ。
// 成功時に `errors: []` / `messages: []` を返すのは、この型の MarshalJSON が nil を [] に揃えて
// いるからで（2026-09-15 まで null だった。ADR-0710・外部レビュー #8）、素の
// `[]*message.GkillError` で書いた応答型だけが `errors: null` へ静かに戻る。
// コンパイルも既存テスト（req_res_test.go はキー名しか見ない）も通るので、型をソース走査で固定する。
func TestResponseErrorAndMessageFieldsUseNamedSliceTypes(t *testing.T) {
	files, err := filepath.Glob("*_response.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) < 80 {
		t.Fatalf("*_response.go が %d 本しか無い（置き場所が変わった可能性がある）", len(files))
	}

	want := map[string]string{"Errors": "GkillErrors", "Messages": "GkillMessages"}
	checked := map[string]int{}
	for _, file := range files {
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		ast.Inspect(parsed, func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					wantType, ok := want[name.Name]
					if !ok {
						continue
					}
					checked[name.Name]++
					sel, ok := field.Type.(*ast.SelectorExpr)
					pkg, _ := sel.X.(*ast.Ident)
					if !ok || pkg == nil || pkg.Name != "message" || sel.Sel.Name != wantType {
						t.Errorf("%s: %s の型は message.%s にすること（素のスライスだと成功時に null が返る）", fset.Position(field.Pos()), name.Name, wantType)
					}
					if field.Tag == nil || !strings.Contains(field.Tag.Value, `json:"`+strings.ToLower(name.Name)+`"`) {
						t.Errorf("%s: %s の json タグは %q にすること", fset.Position(field.Pos()), name.Name, strings.ToLower(name.Name))
					}
				}
			}
			return true
		})
	}
	for name := range want {
		if checked[name] < 80 {
			t.Errorf("%s フィールドを %d 箇所しか検査していない（応答型の形が変わった可能性がある）", name, checked[name])
		}
	}
}
