package usecase

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// このファイルは「規約から外れた書き方が再び現れていないか」をソース走査で見張る。
// どれも、外れても go build も go vet も通り、実行時にエラーも出ずに
// 静かに間違った結果を返す種類のズレなので、機械検査でしか気付けない。

const (
	// 走査の根。usecase パッケージから見た gkill ディレクトリ
	sourceScanGkillRoot = ".."
	// go.mod のある src/server
	sourceScanModuleRoot = "../.."
	sourceScanModulePath = "github.com/mt3hr/gkill/src/server"
)

// walkGoFiles は root 配下の .go ファイルを1つずつ fn へ渡す。
// includeTest=false ならテストファイルは渡さない。
// 「走査対象が0件でも違反なしで通る」のを防ぐため、読めたファイル数を返す。
func walkGoFiles(t *testing.T, root string, includeTest bool, fn func(path string, content string)) int {
	t.Helper()
	scanned := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		if !includeTest && strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		fn(filepath.ToSlash(path), string(content))
		return nil
	})
	if err != nil {
		t.Fatalf("%s の走査に失敗した: %v", root, err)
	}
	if scanned < 100 {
		t.Fatalf("走査できたGoファイルが %d 件しかない。rootの指定(%s)が間違っている可能性がある", scanned, root)
	}
	return scanned
}

// 書き込み後のキャッシュ反映のエラーを `_ =` で捨てている箇所。
// 反映が落ちても呼び出し元には何も伝わらず、次の UpdateCache まで最大1分
// 古い応答が見え続ける（その間にPWAが焼き付けると恒久的に古いまま）。
// KFTL の13箇所がこの形だったので、wrap + slog へ揃えたうえで見張る。
var discardedWriteThroughPattern = regexp.MustCompile(`_\s*=\s*[\w.]*\.WriteThrough\w*Cache\(`)

func TestWriteThroughCacheErrorIsNotDiscarded(t *testing.T) {
	violations := []string{}
	walkGoFiles(t, sourceScanGkillRoot, false, func(path string, content string) {
		for i, line := range strings.Split(content, "\n") {
			if discardedWriteThroughPattern.MatchString(line) {
				violations = append(violations, fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimSpace(line)))
			}
		}
	})
	if len(violations) != 0 {
		t.Fatalf("write-through のエラーを握り潰している。wrap して slog へ出すこと:\n%s", strings.Join(violations, "\n"))
	}
}

// PeriodOfTimeWeekOfDays を nil ガード無しで len() する箇所。
//
// UsePeriodOfTime 相当の判定は「開始/終了時刻のどちらかが入っていれば真」なので、
// 曜日を1つも指定していない（= nil）ときにも曜日フィルタの分岐へ入る。
// そこで `len(...) == 7` だけを見ると nil が「7曜日ではない」と判定され、
// 許可曜日が1つも立たずに全件が消える。エラーは出ない。
// find_filter.go / sqlite3impl_util.go / git_commit_log_repository_local_dir_impl.go の
// 3実装すべてで踏んだ罠なので、4箇所目が生まれないよう見張る。
var weekOfDaysLenPattern = regexp.MustCompile(`len\([\w.]*PeriodOfTimeWeekOfDays\)`)

func TestPeriodOfTimeWeekOfDaysHasNilGuard(t *testing.T) {
	violations := []string{}
	walkGoFiles(t, sourceScanGkillRoot, false, func(path string, content string) {
		if !weekOfDaysLenPattern.MatchString(content) {
			return
		}
		// 同じファイルの中で nil と比べていること。
		// 行単位で見ると `if q.X == nil { ... } else if len(q.X) == 0 {` の形が引っかかるので、
		// 判定はファイル単位にしてある。
		if strings.Contains(content, "PeriodOfTimeWeekOfDays == nil") ||
			strings.Contains(content, "PeriodOfTimeWeekOfDays != nil") {
			return
		}
		violations = append(violations, path)
	})
	if len(violations) != 0 {
		t.Fatalf("PeriodOfTimeWeekOfDays を nil ガード無しで len() している。nil=曜日制限なし を先に外すこと:\n%s",
			strings.Join(violations, "\n"))
	}
}

// どこからも import されていないパッケージ。
//
// dao/reps/cache/rep_cache_updater/ が dao/reps/rep_cache_updater/ の
// バイト単位の複製として放置され、片方だけテストが薄いまま二重管理されていた。
// go build も go vet も通るので、到達可能性は別途数えるしかない。
var (
	// 外部モジュール（src/plugins/ 配下の各プラグイン）から import されるので、
	// 本体から到達できなくても正当なパッケージ。
	reachabilityAllowedPackages = map[string]bool{
		sourceScanModulePath + "/gkill/plugin/sdk": true,
	}
	moduleImportPattern = regexp.MustCompile(`"` + regexp.QuoteMeta(sourceScanModulePath) + `[^"]*"`)
)

func TestNoUnreachablePackages(t *testing.T) {
	moduleRoot, err := filepath.Abs(sourceScanModuleRoot)
	if err != nil {
		t.Fatalf("モジュールルートの解決に失敗した: %v", err)
	}

	imported := map[string]bool{}
	// パッケージのあるディレクトリ -> main パッケージかどうか
	packageDirs := map[string]bool{}

	// import の収集はテストファイルも含める（テストからしか使わないパッケージも「生きている」）。
	walkGoFiles(t, sourceScanModuleRoot, true, func(path string, content string) {
		for _, m := range moduleImportPattern.FindAllString(content, -1) {
			imported[strings.Trim(m, `"`)] = true
		}
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		dir := filepath.Dir(abs)
		if _, ok := packageDirs[dir]; !ok {
			packageDirs[dir] = false
		}
		if regexp.MustCompile(`(?m)^package main\b`).MatchString(content) {
			packageDirs[dir] = true
		}
	})

	violations := []string{}
	for dir, isMain := range packageDirs {
		if isMain || dir == moduleRoot {
			continue // エントリポイントとモジュール直下は import されなくて当然
		}
		rel, err := filepath.Rel(moduleRoot, dir)
		if err != nil {
			continue
		}
		importPath := sourceScanModulePath + "/" + filepath.ToSlash(rel)
		if reachabilityAllowedPackages[importPath] || imported[importPath] {
			continue
		}
		violations = append(violations, importPath)
	}
	if len(violations) != 0 {
		t.Fatalf("どこからも import されていないパッケージがある（複製の置き去りを疑うこと）:\n%s",
			strings.Join(violations, "\n"))
	}
}

// 集約リポジトリの Find 系入口がIDリストを分割していない箇所。
//
// FindQuery.IDs は各repのSQLへ `ID IN (?, ...)` として展開されるので、
// 件数無制限のIDリストを渡すと SQLite のバインド変数上限(32766)を超える。
// **超えたときは GkillError が立たず「HTTP 200・errors:null・0件」で返る**ため、
// 失敗したことが呼び出し元にも利用者にも伝わらない。
var (
	aggregateFindSignature = regexp.MustCompile(`^func \(\w+ \*?(\w+Repositories)\) (Find\w+)\(ctx context\.Context, query \*find\.FindQuery\) \(\[\]\w+, error\) \{`)
	// FindKyous はマップを返すので上のシグネチャには当たらない（別途 findKyous 側で分割済み）
)

func TestAggregateFindChunksIDs(t *testing.T) {
	repsDir := filepath.Join(sourceScanGkillRoot, "dao", "reps")
	entries, err := os.ReadDir(repsDir)
	if err != nil {
		t.Fatalf("%s を読めない: %v", repsDir, err)
	}
	violations := []string{}
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_repositories.go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(repsDir, name))
		if err != nil {
			t.Fatalf("%s を読めない: %v", name, err)
		}
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			m := aggregateFindSignature.FindStringSubmatch(strings.TrimRight(line, "\r"))
			if m == nil {
				continue
			}
			checked++
			// 入口の本体（次の `}` まで）に findChunkedByIDs があること
			body := []string{}
			for j := i + 1; j < len(lines) && strings.TrimRight(lines[j], "\r") != "}"; j++ {
				body = append(body, lines[j])
			}
			if !strings.Contains(strings.Join(body, "\n"), "findChunkedByIDs") {
				violations = append(violations, fmt.Sprintf("%s:%d: %s.%s", name, i+1, m[1], m[2]))
			}
		}
	}
	if checked < 10 {
		t.Fatalf("集約リポジトリの Find が %d 件しか見つからない。正規表現がずれている可能性がある", checked)
	}
	if len(violations) != 0 {
		t.Fatalf("IDリストを分割せずに検索している入口がある。findChunkedByIDs を通すこと:\n%s",
			strings.Join(violations, "\n"))
	}
}
