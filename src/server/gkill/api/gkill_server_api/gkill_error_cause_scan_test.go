package gkill_server_api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GkillError の reason（何が起きたか）は Cause から分類する。
// `if err != nil { ... }` の中で GkillError を組み立てるときに `Cause: err` を付け忘れると、
// そのエラーだけ reason が出ず、gkill_error.log の request failed 行にも cause が載らない。
// 応答は従来どおり返るので目の前ではエラーにならない —— だからソース走査で機械的に検査する。
//
// 2026-09-15 に codemod で 521 箇所へ一斉に付けた。新しいハンドラ・ユースケースも同じ形で書くこと。
// 経緯は documents/adr/0710-error-kind-and-reason-on-the-wire.md。

// causeScanDirs は検査対象（このパッケージと usecase）。
var causeScanDirs = []string{".", filepath.Join("..", "..", "usecase")}

func isErrLikeIdent(e ast.Expr) (string, bool) {
	id, ok := e.(*ast.Ident)
	if !ok {
		return "", false
	}
	if id.Name == "err" || strings.HasSuffix(id.Name, "Err") || strings.HasSuffix(id.Name, "err") {
		return id.Name, true
	}
	return "", false
}

// errNotNilCondIdent は `X != nil`（X が err 系の識別子）なら X の名前を返す。
func errNotNilCondIdent(cond ast.Expr) (string, bool) {
	be, ok := cond.(*ast.BinaryExpr)
	if !ok || be.Op != token.NEQ {
		return "", false
	}
	nilIdent := func(e ast.Expr) bool { id, ok := e.(*ast.Ident); return ok && id.Name == "nil" }
	if name, ok := isErrLikeIdent(be.X); ok && nilIdent(be.Y) {
		return name, true
	}
	if name, ok := isErrLikeIdent(be.Y); ok && nilIdent(be.X) {
		return name, true
	}
	return "", false
}

func isMessageGkillErrorLit(cl *ast.CompositeLit) bool {
	sel, ok := cl.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "message" && sel.Sel.Name == "GkillError"
}

func compositeLitHasKey(cl *ast.CompositeLit, key string) bool {
	for _, el := range cl.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if k, ok := kv.Key.(*ast.Ident); ok && k.Name == key {
			return true
		}
	}
	return false
}

// findGkillErrorLitsWithoutCause は「err 系の if ブロック内にあるのに Cause が無い GkillError リテラル」の位置を集める。
func findGkillErrorLitsWithoutCause(fset *token.FileSet, file *ast.File) []token.Position {
	var missing []token.Position
	var visit func(n ast.Node, inErrBlock bool)
	visit = func(n ast.Node, inErrBlock bool) {
		if n == nil {
			return
		}
		switch v := n.(type) {
		case *ast.IfStmt:
			if v.Init != nil {
				visit(v.Init, inErrBlock)
			}
			visit(v.Cond, inErrBlock)
			_, isErrCond := errNotNilCondIdent(v.Cond)
			visit(v.Body, inErrBlock || isErrCond)
			if v.Else != nil {
				visit(v.Else, inErrBlock)
			}
			return
		case *ast.CompositeLit:
			if isMessageGkillErrorLit(v) && inErrBlock && !compositeLitHasKey(v, "Cause") {
				missing = append(missing, fset.Position(v.Lbrace))
			}
		}
		ast.Inspect(n, func(child ast.Node) bool {
			if child == n {
				return true
			}
			switch child.(type) {
			case *ast.IfStmt, *ast.CompositeLit:
				visit(child, inErrBlock)
				return false
			}
			return true
		})
	}
	for _, decl := range file.Decls {
		visit(decl, false)
	}
	return missing
}

// TestGkillErrorInErrBlockHasCause は、`if err != nil` の中で組み立てる GkillError に
// Cause が付いていることを確認する。
//
// **落ちたら、GkillError リテラルに `Cause: err,` を1行足すこと。**
// err 以外の名前（decodeErr 等）で受けているなら、その識別子を渡す。
func TestGkillErrorInErrBlockHasCause(t *testing.T) {
	checked := 0
	for _, dir := range causeScanDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir %s: %v", dir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(dir, name)
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			checked++
			for _, pos := range findGkillErrorLitsWithoutCause(fset, file) {
				t.Errorf("%s:%d: if err != nil の中の message.GkillError に Cause が無い。`Cause: err,` を足すこと", pos.Filename, pos.Line)
			}
		}
	}
	if checked < 100 {
		t.Fatalf("検査したファイルが %d 本しか無い（テストの置き場所か対象ディレクトリが変わった可能性がある）", checked)
	}
}
