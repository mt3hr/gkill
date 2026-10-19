package reps

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// sql_column_order_scan_test.go は、SQL に書いた列の順と、Go 側で値を受け渡す順の食い違いをソースから探す。
//
//   - SELECT の射影の列順 と rows.Scan の引数の順
//   - INSERT の列リスト と 引数（[]any{...} / ExecContext の可変長引数）の順
//
// SQLite は列の数さえ合っていればエラーも警告も出さないので、順がずれると値が別の列へ黙って入る。
// 通知・リポストの実 rep とキャッシュ版（読むたびに作成者と作成端末が入れ替わる）、Mi の temp rep、
// ファイルの記録の temp rep（INSERT で作成者と作成端末が入れ替わり、連鎖削除のたびに実 rep へ確定する）と
// 同じ形が続けて見つかったので、書いた時点で落とす。振る舞いは audit_fields_all_layers_test.go が固定する。
//
// 照合するのは「両側とも作成・更新の欄や ID などの決まった名前」の位置だけ（sqlScanMetaNames）。
// 名前で判断できない位置（定数・now・型固有の欄）は見ない。構築関数でプリペアして構造体の欄に持ち、
// 別の関数で実行する文（キャッシュ版の追加など）も、欄名を通して辿る。

// sqlScanMetaNames は照合する名前（小文字・アンダースコア抜き・末尾の str / unix を落とした形）。
var sqlScanMetaNames = map[string]bool{
	"createtime": true, "createapp": true, "createdevice": true, "createuser": true,
	"updatetime": true, "updateapp": true, "updatedevice": true, "updateuser": true,
	"relatedtime": true, "id": true, "isdeleted": true, "targetid": true, "repname": true, "datatype": true,
	"txid": true, "userid": true, "device": true,
}

func normalizeSQLScanName(name string) string {
	n := strings.ToLower(strings.ReplaceAll(name, "_", ""))
	for _, suffix := range []string{"unix", "string", "str"} {
		if strings.HasSuffix(n, suffix) && len(n) > len(suffix)+2 {
			return strings.TrimSuffix(n, suffix)
		}
	}
	return n
}

// sqlScanMismatches は列名と引数名を位置ごとに比べ、両方が sqlScanMetaNames にあって違う位置を返す。
func sqlScanMismatches(columns []string, args []string) []string {
	mismatches := []string{}
	for i := range columns {
		c, a := normalizeSQLScanName(columns[i]), normalizeSQLScanName(args[i])
		if c != a && sqlScanMetaNames[c] && sqlScanMetaNames[a] {
			mismatches = append(mismatches, columns[i]+" <- "+args[i])
		}
	}
	return mismatches
}

// foldSQLString は文字列リテラルと + の連結を1本の文字列へ畳む。リテラルでない項（表名の QuoteIdent など）は
// " __X__ " に置き換える。リテラルを1つも含まなければ false。
func foldSQLString(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		if err != nil {
			return "", false
		}
		return s, true
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		l, lok := foldSQLString(e.X)
		r, rok := foldSQLString(e.Y)
		if !lok && !rok {
			return "", false
		}
		if !lok {
			l = " __X__ "
		}
		if !rok {
			r = " __X__ "
		}
		return l + r, true
	case *ast.ParenExpr:
		return foldSQLString(e.X)
	}
	return "", false
}

// sqlScanValueName は引数の式から「どの値か」を表す名前を取る。
// &kmemo.CreateDevice → CreateDevice、kmemo.CreateTime.Format(...) / .Unix() → CreateTime、createTimeStr → createTimeStr。
func sqlScanValueName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.UnaryExpr:
		return sqlScanValueName(e.X)
	case *ast.StarExpr:
		return sqlScanValueName(e.X)
	case *ast.ParenExpr:
		return sqlScanValueName(e.X)
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.CallExpr:
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if _, isPkg := sel.X.(*ast.Ident); !isPkg || len(e.Args) == 0 {
				// メソッド呼び出し（x.CreateTime.Format(...)）は受け手の名前
				return sqlScanValueName(sel.X)
			}
		}
		if len(e.Args) > 0 {
			return sqlScanValueName(e.Args[0])
		}
	case *ast.BasicLit:
		return e.Value
	}
	return ""
}

var (
	sqlScanSelectRe = regexp.MustCompile(`(?is)\bSELECT\b(\s+DISTINCT\b)?`)
	sqlScanInsertRe = regexp.MustCompile(`(?is)INSERT\s+(?:OR\s+\w+\s+)?INTO\s+\S+\s*\(([^)]*)\)\s*VALUES\s*\(`)
	sqlScanAliasRe  = regexp.MustCompile(`(?is)\bAS\s+"?(\w+)"?\s*$`)
	sqlScanLastRe   = regexp.MustCompile(`(\w+)\s*$`)
)

// sqlScanSplitTop はかっこの外のカンマで分ける。
func sqlScanSplitTop(s string) []string {
	out := []string{}
	depth := 0
	start := 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if rest := strings.TrimSpace(s[start:]); rest != "" {
		out = append(out, rest)
	}
	return out
}

// sqlScanProjections は SQL の中の SELECT ごとに、かっこの外の FROM までの列名を返す（副問い合わせも含む）。
func sqlScanProjections(sql string) [][]string {
	projections := [][]string{}
	upper := strings.ToUpper(sql)
	for _, loc := range sqlScanSelectRe.FindAllStringIndex(sql, -1) {
		depth := 0
		end := len(sql)
	scan:
		for i := loc[1]; i < len(sql); i++ {
			switch sql[i] {
			case '(':
				depth++
			case ')':
				depth--
				if depth < 0 {
					end = i
					break scan
				}
			default:
				if depth == 0 && strings.HasPrefix(upper[i:], "FROM") && i > 0 && !isSQLScanIdentByte(sql[i-1]) &&
					(i+4 >= len(sql) || !isSQLScanIdentByte(sql[i+4])) {
					end = i
					break scan
				}
			}
		}
		columns := []string{}
		for _, item := range sqlScanSplitTop(sql[loc[1]:end]) {
			if m := sqlScanAliasRe.FindStringSubmatch(item); m != nil {
				columns = append(columns, m[1])
			} else if m := sqlScanLastRe.FindStringSubmatch(item); m != nil {
				columns = append(columns, m[1])
			} else {
				columns = append(columns, item)
			}
		}
		projections = append(projections, columns)
	}
	return projections
}

func isSQLScanIdentByte(b byte) bool {
	return b == '_' || ('0' <= b && b <= '9') || ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}

func sqlScanInsertColumns(sql string) []string {
	m := sqlScanInsertRe.FindStringSubmatch(sql)
	if m == nil {
		return nil
	}
	columns := []string{}
	for _, c := range strings.Split(m[1], ",") {
		if c = strings.TrimSpace(c); c != "" {
			columns = append(columns, c)
		}
	}
	return columns
}

func hasSQLScanMetaColumn(columns []string) bool {
	for _, c := range columns {
		if sqlScanMetaNames[normalizeSQLScanName(c)] {
			return true
		}
	}
	return false
}

// sqlScanFunc は関数1つから集めたもの。
type sqlScanFunc struct {
	name     string
	sqls     []string            // 畳んだ SQL
	sqlVars  map[string]string   // 変数名 → SQL（最後の代入）
	scans    [][]string          // rows.Scan の引数名
	argLists [][]string          // []any{...} と Exec / Query の引数名
	argVars  map[string][]string // 変数名 → []any{...} の中身
	// stmtCalls は x.<欄>.ExecContext / QueryContext / QueryRowContext(ctx, v...) の (欄, v)
	stmtCalls [][2]string
}

func isSQLScanAnySliceType(expr ast.Expr) bool {
	arr, ok := expr.(*ast.ArrayType)
	if !ok || arr.Len != nil {
		return false
	}
	switch elt := arr.Elt.(type) {
	case *ast.Ident:
		return elt.Name == "any"
	case *ast.InterfaceType:
		return true
	}
	return false
}

var sqlScanExecNames = map[string]bool{"ExecContext": true, "QueryContext": true, "QueryRowContext": true, "Exec": true, "Query": true, "QueryRow": true}

// collectSQLScanFunc は関数本体から SQL・Scan・引数並び・プリペアした文の代入を集める。
// stmtVarSQL（変数名 → SQL）は PrepareContext の戻りを受けた変数で、ファイル全体で共有する。
func collectSQLScanFunc(fn *ast.FuncDecl, stmtVarSQL map[string]string, stmtFieldSQL map[string]string) *sqlScanFunc {
	f := &sqlScanFunc{name: fn.Name.Name, sqlVars: map[string]string{}, argVars: map[string][]string{}}
	valueNames := func(exprs []ast.Expr) []string {
		names := []string{}
		for _, e := range exprs {
			names = append(names, sqlScanValueName(e))
		}
		return names
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if len(x.Lhs) == len(x.Rhs) {
				for i, rhs := range x.Rhs {
					id, ok := x.Lhs[i].(*ast.Ident)
					if !ok {
						continue
					}
					if s, ok := foldSQLString(rhs); ok && x.Tok != token.ADD_ASSIGN {
						f.sqlVars[id.Name] = s
					}
					if lit, ok := rhs.(*ast.CompositeLit); ok && isSQLScanAnySliceType(lit.Type) {
						f.argVars[id.Name] = valueNames(lit.Elts)
					}
				}
			}
			// stmt, err := db.PrepareContext(ctx, sqlVar)
			if len(x.Rhs) == 1 && len(x.Lhs) >= 1 {
				if call, ok := x.Rhs[0].(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "PrepareContext" && len(call.Args) == 2 {
						if lhs, ok := x.Lhs[0].(*ast.Ident); ok {
							if sqlID, ok := call.Args[1].(*ast.Ident); ok {
								if s, ok := f.sqlVars[sqlID.Name]; ok {
									stmtVarSQL[lhs.Name] = s
								}
							} else if s, ok := foldSQLString(call.Args[1]); ok {
								stmtVarSQL[lhs.Name] = s
							}
						}
					}
				}
			}
			// recv.field = stmtVar
			for i, lhs := range x.Lhs {
				if sel, ok := lhs.(*ast.SelectorExpr); ok && i < len(x.Rhs) {
					if id, ok := x.Rhs[i].(*ast.Ident); ok {
						if s, ok := stmtVarSQL[id.Name]; ok {
							stmtFieldSQL[sel.Sel.Name] = s
						}
					}
				}
			}
		case *ast.KeyValueExpr:
			// &impl{field: stmtVar}
			if key, ok := x.Key.(*ast.Ident); ok {
				if id, ok := x.Value.(*ast.Ident); ok {
					if s, ok := stmtVarSQL[id.Name]; ok {
						stmtFieldSQL[key.Name] = s
					}
				}
			}
		case *ast.BasicLit, *ast.BinaryExpr:
			if s, ok := foldSQLString(x.(ast.Expr)); ok {
				f.sqls = append(f.sqls, s)
				return false
			}
		case *ast.CompositeLit:
			if isSQLScanAnySliceType(x.Type) {
				f.argLists = append(f.argLists, valueNames(x.Elts))
			}
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name == "Scan" {
				f.scans = append(f.scans, valueNames(x.Args))
				return true
			}
			if !sqlScanExecNames[sel.Sel.Name] {
				return true
			}
			if x.Ellipsis.IsValid() {
				// x.<欄>.ExecContext(ctx, queryArgs...)
				if inner, ok := sel.X.(*ast.SelectorExpr); ok && len(x.Args) > 0 {
					if v, ok := x.Args[len(x.Args)-1].(*ast.Ident); ok {
						f.stmtCalls = append(f.stmtCalls, [2]string{inner.Sel.Name, v.Name})
					}
				}
				return true
			}
			args := x.Args
			if len(args) > 0 {
				if id, ok := args[0].(*ast.Ident); ok && id.Name == "ctx" {
					args = args[1:]
				}
			}
			if len(args) > 0 {
				if _, ok := foldSQLString(args[0]); ok {
					args = args[1:]
				} else if id, ok := args[0].(*ast.Ident); ok {
					if _, isSQL := f.sqlVars[id.Name]; isSQL {
						args = args[1:]
					}
				}
			}
			if len(args) > 0 {
				f.argLists = append(f.argLists, valueNames(args))
			}
		}
		return true
	})
	return f
}

type sqlScanStats struct {
	selects int // 照合できた SELECT の射影（作成・更新の欄などを含むもの）
	inserts int // 照合できた INSERT
}

// checkSQLScanAgainst は列名を同じ長さの候補と比べる。候補が無ければ checked=false。
// 候補のどれか1つでも食い違いが無ければよい（同じ関数に別の文の Scan がある）。
func checkSQLScanAgainst(columns []string, candidates [][]string) (checked bool, mismatches []string) {
	for _, cand := range candidates {
		if len(cand) != len(columns) {
			continue
		}
		m := sqlScanMismatches(columns, cand)
		if len(m) == 0 {
			return true, nil
		}
		if !checked {
			mismatches = m
		}
		checked = true
	}
	return checked, mismatches
}

// scanSQLColumnOrder は dir 直下の Go ファイル（テストを除く）を走査して違反を返す。
func scanSQLColumnOrder(t *testing.T, dir string) ([]string, sqlScanStats, int) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir %s: %v", dir, err)
	}
	violations := []string{}
	stats := sqlScanStats{}
	scannedFiles := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", path, err)
		}
		scannedFiles++

		stmtVarSQL := map[string]string{}
		stmtFieldSQL := map[string]string{}
		funcs := []*sqlScanFunc{}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				funcs = append(funcs, collectSQLScanFunc(fn, stmtVarSQL, stmtFieldSQL))
			}
		}

		report := func(f *sqlScanFunc, kind string, columns []string, mismatches []string) {
			violations = append(violations, name+":"+f.name+": "+kind+" の列順と Go 側の順が食い違う "+strings.Join(mismatches, ", ")+
				"（列: "+strings.Join(columns, ", ")+"）")
		}
		for _, f := range funcs {
			for _, sql := range f.sqls {
				for _, columns := range sqlScanProjections(sql) {
					if !hasSQLScanMetaColumn(columns) {
						continue
					}
					if checked, mismatches := checkSQLScanAgainst(columns, f.scans); checked {
						stats.selects++
						if len(mismatches) != 0 {
							report(f, "SELECT と rows.Scan", columns, mismatches)
						}
					}
				}
				if columns := sqlScanInsertColumns(sql); columns != nil && hasSQLScanMetaColumn(columns) {
					if checked, mismatches := checkSQLScanAgainst(columns, f.argLists); checked {
						stats.inserts++
						if len(mismatches) != 0 {
							report(f, "INSERT の列と引数", columns, mismatches)
						}
					}
				}
			}
			// 構築関数でプリペアし、構造体の欄に持って別の関数で実行する文
			for _, call := range f.stmtCalls {
				sql, ok := stmtFieldSQL[call[0]]
				if !ok {
					continue
				}
				if columns := sqlScanInsertColumns(sql); columns != nil && hasSQLScanMetaColumn(columns) {
					if args, ok := f.argVars[call[1]]; ok {
						if checked, mismatches := checkSQLScanAgainst(columns, [][]string{args}); checked {
							stats.inserts++
							if len(mismatches) != 0 {
								report(f, "INSERT の列と引数（"+call[0]+"）", columns, mismatches)
							}
						}
					}
					continue
				}
				for _, columns := range sqlScanProjections(sql) {
					if !hasSQLScanMetaColumn(columns) {
						continue
					}
					if checked, mismatches := checkSQLScanAgainst(columns, f.scans); checked {
						stats.selects++
						if len(mismatches) != 0 {
							report(f, "SELECT と rows.Scan（"+call[0]+"）", columns, mismatches)
						}
					}
				}
			}
		}
	}
	return violations, stats, scannedFiles
}

// TestSQLColumnOrderMatchesGoOrder は dao/reps と dao/reps/cache の SQL を走査し、列の順と
// Scan / 引数の順の食い違いが復活したら落とす。
func TestSQLColumnOrderMatchesGoOrder(t *testing.T) {
	total := sqlScanStats{}
	for _, dir := range []string{".", "cache"} {
		violations, stats, scannedFiles := scanSQLColumnOrder(t, dir)
		if scannedFiles == 0 {
			t.Fatalf("%s で走査できた Go ファイルが0件。カレントディレクトリの指定が間違っている可能性がある", dir)
		}
		if len(violations) != 0 {
			t.Errorf("%s: SQL の列順と Go 側の順が食い違っている（SQLite はエラーを出さずに別の列へ入れる）:\n  %s", dir, strings.Join(violations, "\n  "))
		}
		total.selects += stats.selects
		total.inserts += stats.inserts
	}
	t.Logf("照合した文: SELECT 射影 %d / INSERT %d", total.selects, total.inserts)
	// 照合が空振りして「違反なし」で通るのを防ぐ。下限は実測（テスト由来: SELECT 射影 250 / INSERT 53）から少し下げた値。
	// 文を足して増えるのはよい。大きく減ったら、走査の取りこぼし（書き方が変わって辿れなくなった）を疑う。
	if total.selects < 240 || total.inserts < 50 {
		t.Fatalf("照合できた文が少なすぎる: SELECT 射影 %d / INSERT %d（走査が文を辿れなくなっていないか）", total.selects, total.inserts)
	}
}

// TestSQLColumnOrderScanDetectsSwaps は、走査が実際に入れ替わりを見つけることを合成ソースで確かめる。
func TestSQLColumnOrderScanDetectsSwaps(t *testing.T) {
	dir := t.TempDir()
	src := "package fixture\n\n" +
		"func swappedInsert(x X) {\n" +
		"\tsql := `INSERT INTO IDF (ID, CREATE_USER, CREATE_DEVICE) VALUES (?, ?, ?)`\n" +
		"\tqueryArgs := []any{x.ID, x.CreateDevice, x.CreateUser}\n" +
		"\t_, _ = db.ExecContext(ctx, sql, queryArgs...)\n" +
		"}\n\n" +
		"func swappedSelect() {\n" +
		"\tsql := `SELECT ID, CREATE_USER, CREATE_DEVICE FROM NOTIFICATION WHERE ID = ?`\n" +
		"\t_ = rows.Scan(&n.ID, &n.CreateDevice, &n.CreateUser)\n" +
		"}\n\n" +
		"func newRep() *rep {\n" +
		"\taddSQL := `INSERT INTO ` + quote(name) + ` (ID, UPDATE_DEVICE, UPDATE_USER, CREATE_TIME_UNIX) VALUES (?, ?, ?, ?)`\n" +
		"\taddStmt, err := db.PrepareContext(ctx, addSQL)\n" +
		"\t_ = err\n" +
		"\treturn &rep{addInfoStmt: addStmt}\n" +
		"}\n\n" +
		"func (r *rep) add(k K) {\n" +
		"\tqueryArgs := []any{k.ID, k.UpdateUser, k.UpdateDevice, k.CreateTime.Unix()}\n" +
		"\t_, _ = r.addInfoStmt.ExecContext(ctx, queryArgs...)\n" +
		"}\n\n" +
		"func correct(x X) {\n" +
		"\tsql := `SELECT ID, CREATE_DEVICE, CREATE_USER, ? AS REP_NAME FROM KMEMO`\n" +
		"\t_ = rows.Scan(&x.ID, &x.CreateDevice, &x.CreateUser, &x.RepName)\n" +
		"\tinsertSQL := `INSERT INTO KMEMO (ID, CREATE_TIME, CREATE_DEVICE, CREATE_USER) VALUES (?, ?, ?, ?)`\n" +
		"\tqueryArgs := []any{x.ID, x.CreateTime.Format(layout), x.CreateDevice, x.CreateUser}\n" +
		"\t_, _ = db.ExecContext(ctx, insertSQL, queryArgs...)\n" +
		"}\n"
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(src), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	violations, stats, _ := scanSQLColumnOrder(t, dir)
	want := []string{"swappedInsert", "swappedSelect", "add"}
	if len(violations) != len(want) {
		t.Fatalf("違反の数 = %d, want %d:\n  %s", len(violations), len(want), strings.Join(violations, "\n  "))
	}
	for i, fn := range want {
		if !strings.Contains(violations[i], ":"+fn+":") {
			t.Errorf("violations[%d] = %q, want 関数 %s の違反", i, violations[i], fn)
		}
	}
	if stats.selects != 2 || stats.inserts != 3 {
		t.Errorf("照合した文 = SELECT %d / INSERT %d, want 2 / 3", stats.selects, stats.inserts)
	}
}
