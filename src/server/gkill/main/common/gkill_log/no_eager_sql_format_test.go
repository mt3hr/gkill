package gkill_log

// TRACE_SQLログの引数が「レベル無効でも必ず組み立てられる」形へ戻ったら落とすソース走査テスト。
//
// Goは引数を呼び出し前に評価するので、
//
//	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), ...)
//
// と素に書くと、既定のログレベル(none)でもSQL全文の%qエスケープと
// 引数のreflect整形が必ず走る。1行ごとのループ(キャッシュ再構築のINSERT)では
// 行数ぶん積み上がり、実データのフル再構築ではGB級のゴミになる。
//
// 正しい書き方は次のどちらか:
//   - 共通ヘルパ(gkill_log.LogSQL / LogSQLParams / LogSQLQuery / LogSQLQueryArgs / LogIndexSQL)を使う
//   - ヘルパの形に当てはまらないものは `if gkill_log.TraceSQLEnabled(ctx) { ... }` で囲む
//
// 作法は usecase/write_through_cache_test.go の TestNoRepsCountCacheGuard に合わせてある。

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoEagerTraceSQLFormatting(t *testing.T) {
	// このファイルは src/server/gkill/main/common/gkill_log/ にあるので、3つ上が gkill/
	root := filepath.Join("..", "..", "..")
	violations := []string{}
	scanned := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		// ヘルパ自身は当然 fmt.Sprintf を持つので対象外
		if strings.Contains(filepath.ToSlash(path), "/main/common/gkill_log/") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
		for i, line := range lines {
			if !strings.Contains(line, "gkill_log.TraceSQL,") || !strings.Contains(line, "fmt.Sprintf") {
				continue
			}
			previous := ""
			if i > 0 {
				previous = strings.TrimSpace(lines[i-1])
			}
			if previous == "if gkill_log.TraceSQLEnabled(ctx) {" {
				continue
			}
			violations = append(violations, fmt.Sprintf("%s:%d: %s", filepath.ToSlash(path), i+1, strings.TrimSpace(line)))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk %s: %v", root, err)
	}
	if scanned == 0 {
		t.Fatalf("走査対象が1ファイルも無い。rootの指定(%s)が間違っている", root)
	}
	if len(violations) != 0 {
		t.Errorf("TRACE_SQLログの引数がレベル無効でも組み立てられる形になっている(%d件):\n  %s\n"+
			"gkill_log.LogSQL系のヘルパを使うか、if gkill_log.TraceSQLEnabled(ctx) で囲むこと",
			len(violations), strings.Join(violations, "\n  "))
	}
}
