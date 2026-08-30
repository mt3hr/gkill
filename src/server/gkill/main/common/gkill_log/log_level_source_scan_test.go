package gkill_log

// ログのレベルが内容と合っていることを機械で確かめるソース走査テスト。
//
// レベルの規約は ADR-1001。要点は2つだけ:
//
//  1. そのエラーが呼び出し元へ返るなら Debug でよい（応答の errors に載り、
//     writeErrorStatus が境界で1行出す）。返らない＝握り潰すなら Debug 禁止。
//  2. 原因が利用者側（入力・認証・認可）なら Warn 以下、サーバ側なら Error。
//
// 破っても目の前ではエラーにならない。既定のログレベルは error なので、
// 深部の Debug は1行も出ない ——「500 が返るのに理由がどこにも出ない」に戻る。
//
// 作法は no_eager_sql_format_test.go / response_status_guard_test.go に合わせてある。

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// scanGkillGoFiles は gkill/ 配下の非テスト .go を1ファイルずつ行配列で渡す。
func scanGkillGoFiles(t *testing.T, visit func(path string, lines []string)) {
	t.Helper()
	// このファイルは src/server/gkill/main/common/gkill_log/ にあるので、3つ上が gkill/
	root := filepath.Join("..", "..", "..")
	scanned := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		if strings.Contains(filepath.ToSlash(path), "/main/common/gkill_log/") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		visit(filepath.ToSlash(path), strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n"))
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk %s: %v", root, err)
	}
	if scanned == 0 {
		t.Fatalf("走査対象が1ファイルも無い。rootの指定(%s)が間違っている", root)
	}
}

// TestNoBareErrorLogMessage は、メッセージが "error" だけの呼び出しが増えていないことを見る。
//
// **落ちたら、ログのメッセージに操作名が入っていない。**
// 以前は703件がこの形で、実際の情報は fmt.Errorf で包んだ文字列側にしか無く、
// メッセージでの集計もアラートも作れなかった。直前の
// err = fmt.Errorf("error at XXX ...: %w", err) の "error at XXX" をメッセージへ移すこと。
func TestNoBareErrorLogMessage(t *testing.T) {
	bareMessages := []string{
		`gkill_log.Debug, "error",`,
		`gkill_log.Info, "error",`,
		`gkill_log.Warn, "error",`,
		`gkill_log.Error, "error",`,
	}
	violations := []string{}
	scanGkillGoFiles(t, func(path string, lines []string) {
		for i, line := range lines {
			for _, bare := range bareMessages {
				if strings.Contains(line, bare) {
					violations = append(violations, fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimSpace(line)))
					break
				}
			}
		}
	})
	if len(violations) != 0 {
		t.Errorf("ログのメッセージが \"error\" だけになっている(%d件):\n  %s\n"+
			"直前の fmt.Errorf(\"error at XXX ...\") の操作名をメッセージへ移すこと",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// deferCloseLevelByMessage は defer の Close 失敗をどのレベルで出すかの表。
//
// 解放に失敗しても実害が無いもの(statement / rows / リクエストボディ / 読み取り用ファイル)は
// Debug、リークやデータの欠けにつながるもの(データベース / 書き込み用に開いたファイル)は Warn。
// メッセージに対象を書いてあるのは、あとからレベルと突き合わせられるようにするため。
var deferCloseLevelByMessage = map[string]string{
	"error at defer close statement":             "Debug",
	"error at defer close rows":                  "Debug",
	"error at defer close request body":          "Debug",
	"error at defer close file opened for read":  "Debug",
	"error at defer close database":              "Warn",
	"error at defer close file opened for write": "Warn",
	"error at defer close gkill server api":      "Error",
}

// TestDeferCloseLogLevel は defer の Close 失敗のメッセージとレベルの対応を固定する。
//
// **落ちたら、Close の対象とレベルが合っていない。**
// 941箇所すべてが同じ "error at defer close" だった頃は、レベルを上げ下げしても
// どの対象に効いているのか誰にも分からなかった。
func TestDeferCloseLogLevel(t *testing.T) {
	violations := []string{}
	seen := map[string]int{}
	scanGkillGoFiles(t, func(path string, lines []string) {
		for i, line := range lines {
			if !strings.Contains(line, "error at defer close") {
				continue
			}
			matched := false
			for msg, want := range deferCloseLevelByMessage {
				if !strings.Contains(line, "\""+msg+"\"") {
					continue
				}
				matched = true
				seen[msg]++
				if !strings.Contains(line, "gkill_log."+want+", ") {
					violations = append(violations, fmt.Sprintf("%s:%d: %q は %s のはず: %s",
						path, i+1, msg, want, strings.TrimSpace(line)))
				}
				break
			}
			if !matched {
				violations = append(violations, fmt.Sprintf("%s:%d: 表に無いdefer closeのメッセージ: %s",
					path, i+1, strings.TrimSpace(line)))
			}
		}
	})
	if len(violations) != 0 {
		t.Errorf("defer の Close ログのレベルが表と合っていない(%d件):\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
	// 走査が空振りしていないことの確認(メッセージの書き方を変えたときに気づく)
	if seen["error at defer close statement"] < 100 {
		t.Errorf("statement の defer close が %d 件しか見つからない。メッセージの書き方が変わった可能性がある",
			seen["error at defer close statement"])
	}
}

// swallowedDebugAllowlist は「呼び出し元へ返さないが Debug のままでよい」ログ。
//
// いずれも常態化しうるか、失敗しても次の機会に取り直せるもの。
// **ここへ足すときは理由を書くこと。** 足せば検査を素通りできてしまうので、
// 「返らないエラーは Debug 禁止」の防御線がここの運用だけになる。
var swallowedDebugAllowlist = map[string]string{
	"error at defer close statement":            "解放失敗に実害なし(deferCloseLevelByMessage が別に見ている)",
	"error at defer close rows":                 "同上",
	"error at defer close request body":         "同上",
	"error at defer close file opened for read": "同上",
	"error at close ffmpeg output":              "読み取り用の一時ファイル",
	"error at close response body":              "HTTPレスポンスボディ",
	"error at fetch attached data for mcp":      "件数を数えて別途まとめて返している",
	"error at get private ipv4 addresses":       "画面表示用の best effort。取れなくても動く",
	"error at get global ip":                    "同上",
	"error at write thumb failed marker":        "次回の生成で取り直せる",
	"error at write video compat failed marker": "同上",
	"failed to fill favicon":                    "外部サイトの取得。落ちるのが常態",
	"failed to fill title to urlog":             "同上",
	"failed to fill description to urlog":       "同上",
	"failed to fill image to urlog":             "同上",
	"plugin exited with error":                  "正常な停止でも出る",
}

// TestSwallowedErrorsAreNotDebug は、握り潰すエラーが Debug のまま増えていないことを見る。
//
// 判定は「そのログが if ブロックの最後の文か」。最後なら return が無い＝
// 呼び出し元へ返らないので、そのログが唯一の記録になる。
//
// **落ちたら、返らないエラーを Debug で出している。**
// 既定のログレベルでは1行も残らないので、失敗したことが誰にも見えない。
// Warn(結果が痩せる)か Error(データが壊れる・機能が止まる)へ上げること。
// 常態化しうるものだけ、理由を添えて swallowedDebugAllowlist へ足す。
func TestSwallowedErrorsAreNotDebug(t *testing.T) {
	violations := []string{}
	scanGkillGoFiles(t, func(path string, lines []string) {
		for i, line := range lines {
			if !strings.Contains(line, "gkill_log.Debug,") {
				continue
			}
			// 次の非空行が閉じ括弧だけなら、このログがブロックの最後の文
			next := i + 1
			for next < len(lines) && strings.TrimSpace(lines[next]) == "" {
				next++
			}
			if next >= len(lines) || strings.TrimSpace(lines[next]) != "}" {
				continue
			}
			msg := logMessageOf(line)
			if _, allowed := swallowedDebugAllowlist[msg]; allowed {
				continue
			}
			violations = append(violations, fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimSpace(line)))
		}
	})
	if len(violations) != 0 {
		t.Errorf("呼び出し元へ返らないエラーを Debug で出している(%d件):\n  %s\n"+
			"Warn か Error へ上げるか、理由を添えて swallowedDebugAllowlist へ足すこと",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// logMessageOf は slog.Log(..., gkill_log.X, "msg", ...) の "msg" を取り出す。
func logMessageOf(line string) string {
	const marker = "gkill_log."
	idx := strings.Index(line, marker)
	if idx < 0 {
		return ""
	}
	rest := line[idx:]
	first := strings.Index(rest, "\"")
	if first < 0 {
		return ""
	}
	rest = rest[first+1:]
	msg, _, found := strings.Cut(rest, "\"")
	if !found {
		return ""
	}
	return msg
}

// TestSwallowedDebugAllowlistIsUsed は許可リストの各項目が実在することを見る。
//
// 実在しない項目が残っていると、直したはずの箇所が名前を変えて戻ってきたときに見逃す。
func TestSwallowedDebugAllowlistIsUsed(t *testing.T) {
	used := map[string]bool{}
	scanGkillGoFiles(t, func(_ string, lines []string) {
		for _, line := range lines {
			if !strings.Contains(line, "gkill_log.") {
				continue
			}
			if msg := logMessageOf(line); msg != "" {
				used[msg] = true
			}
		}
	})
	for msg, reason := range swallowedDebugAllowlist {
		if !used[msg] {
			t.Errorf("swallowedDebugAllowlist の %q (%s) を出している箇所が無い。"+
				"メッセージを変えたか消したなら許可リストも直すこと", msg, reason)
		}
	}
}
