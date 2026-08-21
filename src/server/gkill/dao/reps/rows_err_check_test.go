package reps

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestRowsNextLoopsCheckRowsErr は、dao/reps 配下で rows.Next() を回す関数が
// rows.Err() を確認しない状態が復活したら落とす。
//
// database/sql の rows.Next() は反復が途中でエラーになったときも false を返すので、
// ループ後に rows.Err() を見ないと「部分的な結果」を成功として返してしまう。
// GetLatestDataRepositoryAddress ではそれが最新版アドレス一覧の静かな欠落につながる。
func TestRowsNextLoopsCheckRowsErr(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	fset := token.NewFileSet()
	scanned := 0
	violations := []string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("failed to read %s: %v", name, err)
		}
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", name, err)
		}
		scanned++
		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			start := fset.Position(fn.Body.Pos()).Offset
			end := fset.Position(fn.Body.End()).Offset
			body := string(src[start:end])
			if strings.Contains(body, "rows.Next()") && !strings.Contains(body, "rows.Err()") {
				violations = append(violations, name+":"+fn.Name.Name)
			}
			return true
		})
	}
	// 走査対象が0件でも「違反なし」で通ってしまうので、実際に読めたことを確かめる。
	if scanned < 20 {
		t.Fatalf("走査できたGoファイルが %d 件しかない。カレントディレクトリの指定が間違っている可能性がある", scanned)
	}
	if len(violations) != 0 {
		t.Fatalf("rows.Next() を回す関数が rows.Err() を確認していない（反復途中の失敗を握り潰す）:\n  %s", strings.Join(violations, "\n  "))
	}
}
