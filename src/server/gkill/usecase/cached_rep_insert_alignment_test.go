package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// キャッシュrepの INSERT で、列の並びと queryArgs の並びがずれていないかを見る。
//
// SQLite のプレースホルダには名前が無いので、列と引数がずれても**エラーにならない**。
// 型まで同じ文字列カラムどうしだと、値が入れ替わったまま保存され、読み出しでも
// 入れ替わって返る。実DB(leaf)は無傷なので、キャッシュを作り直すまで直らないのに原因も見えない。
//
// 実例: idf_kyou_repository_cached_sqlite3_impl.go は列が
// (CREATE_APP, CREATE_USER, CREATE_DEVICE) なのに引数が
// (CreateApp, CreateDevice, CreateUser) で、**作成ユーザと作成端末が入れ替わって**いた。
// write-through と UpdateCache の両方に同じずれがあり、13種のうちIDFだけが該当
// (2026-08-19 修正)。--cache_in_memory の既定は true なので通常の読み書き経路。
//
// 対応づけは「SQLを入れている識別子」で行う。近くにある引数リストと突き合わせる作り方だと、
// SQLを構造体フィールドに持って遠くで使うIDFのような書き方を取り違える
// （実際、最初にそう書いて**直したはずのずれを検出できなかった**）。
var (
	sqlAssignRe    = regexp.MustCompile(`^\s*(?:(\w+)\s*:?=|(\w+):)\s*` + "`" + `\s*$`)
	insertHeaderRe = regexp.MustCompile(`INSERT INTO`)
	sqlColumnRe    = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	argFieldRe     = regexp.MustCompile(`^&?\w+\.([A-Za-z0-9_]+)(?:\.\w+\(\))?$`)
	prepareRe      = regexp.MustCompile(`PrepareContext\(ctx,\s*(?:i\.|k\.|t\.|m\.|n\.|u\.|l\.|r\.|g\.)?(\w+)\)`)
	execStmtRe     = regexp.MustCompile(`(?:i|k|t|m|n|u|l|r|g)\.(\w+)Stmt\.ExecContext\(ctx,\s*queryArgs`)
)

// camelize は SQL の列名(CREATE_USER)を Go のフィールド名(CreateUser)へ変える
func camelize(column string) string {
	out := ""
	for _, part := range strings.Split(strings.ToLower(column), "_") {
		if part == "" {
			continue
		}
		out += strings.ToUpper(part[:1]) + part[1:]
	}
	return out
}

// isSwapSensitive は「入れ替わると意味が変わるのに型が同じで気付けない」列かどうか
func isSwapSensitive(field string) bool {
	switch field {
	case "CreateUser", "CreateDevice", "CreateApp", "UpdateUser", "UpdateDevice", "UpdateApp":
		return true
	}
	return false
}

func TestCachedRepInsertColumnsMatchArgs(t *testing.T) {
	repsDir := filepath.Join(sourceScanGkillRoot, "dao", "reps")
	entries, err := os.ReadDir(repsDir)
	if err != nil {
		t.Fatalf("%s を読めない: %v", repsDir, err)
	}

	violations := []string{}
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_repository_cached_sqlite3_impl.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(repsDir, name))
		if err != nil {
			t.Fatalf("%s を読めない: %v", name, err)
		}
		lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")

		// 1) 識別子 -> INSERT の列並び
		columnsBySQLName := map[string][]string{}
		for i := 0; i < len(lines); i++ {
			m := sqlAssignRe.FindStringSubmatch(lines[i])
			if m == nil {
				continue
			}
			sqlName := m[1]
			if sqlName == "" {
				sqlName = m[2]
			}
			// SQL は `INSERT INTO ` + QuoteIdent(dbName) + ` (` のように
			// バッククォート文字列を連結して組み立てているので、
			// 「バッククォートが出たら終わり」では1行目で切れてしまう。
			// VALUES までを本文とみなして列名を拾う
			body := []string{}
			for j := i + 1; j < len(lines) && j < i+80; j++ {
				body = append(body, lines[j])
				if strings.Contains(lines[j], "VALUES") {
					break
				}
			}
			if len(body) == 0 || !insertHeaderRe.MatchString(body[0]) {
				continue
			}
			columns := []string{}
			for _, line := range body {
				if strings.Contains(line, "VALUES") {
					break
				}
				token := strings.TrimSuffix(strings.TrimSpace(line), ",")
				if sqlColumnRe.MatchString(token) {
					columns = append(columns, token)
				}
			}
			if len(columns) >= 3 {
				columnsBySQLName[sqlName] = columns
			}
		}

		// 2) queryArgs の塊ごとに、対応する SQL 名を探して突き合わせる
		for i := 0; i < len(lines); i++ {
			if !strings.Contains(lines[i], "queryArgs") || !strings.Contains(lines[i], "[]any{") {
				continue
			}
			args := []string{}
			depth := 1
			end := i
			for j := i + 1; j < len(lines) && depth > 0; j++ {
				line := strings.TrimSpace(lines[j])
				depth += strings.Count(line, "{") - strings.Count(line, "}")
				if depth <= 0 {
					end = j
					break
				}
				token := strings.TrimSuffix(line, ",")
				if token == "" || strings.HasPrefix(token, "//") {
					continue
				}
				if m := argFieldRe.FindStringSubmatch(token); m != nil {
					args = append(args, m[1])
					continue
				}
				args = append(args, "")
			}

			sqlName := findSQLNameForArgs(lines, i, end)
			if sqlName == "" {
				continue
			}
			columns, ok := columnsBySQLName[sqlName]
			if !ok || len(columns) != len(args) {
				continue
			}
			checked++
			for k := range columns {
				expected := camelize(columns[k])
				if args[k] == "" || !isSwapSensitive(expected) {
					continue
				}
				if args[k] != expected {
					violations = append(violations, fmt.Sprintf(
						"%s:%d: %s の %d 番目の列 %s に %s を渡している（%s のはず）",
						name, i+1, sqlName, k+1, columns[k], args[k], expected))
				}
			}
			i = end
		}
	}

	if checked < 10 {
		t.Fatalf("突き合わせられた INSERT が %d 件しかない。抽出のしかたがずれている可能性がある", checked)
	}
	if len(violations) != 0 {
		t.Fatalf("キャッシュrepの INSERT で列と引数がずれている:\n%s", strings.Join(violations, "\n"))
	}
}

// findSQLNameForArgs は queryArgs の塊に対応する SQL の識別子を返す。
//   - 塊の後ろ: `i.xxxStmt.ExecContext(ctx, queryArgs...)` → xxxSQL
//   - 塊の前:   直近の `PrepareContext(ctx, xxx)`
func findSQLNameForArgs(lines []string, start int, end int) string {
	for j := end; j < len(lines) && j < end+12; j++ {
		if m := execStmtRe.FindStringSubmatch(lines[j]); m != nil {
			return m[1] + "SQL"
		}
	}
	for j := start; j >= 0 && j > start-80; j-- {
		if m := prepareRe.FindStringSubmatch(lines[j]); m != nil {
			return m[1]
		}
	}
	return ""
}
