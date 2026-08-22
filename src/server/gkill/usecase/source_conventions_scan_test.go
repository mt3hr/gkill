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

// dao/reps 側で FindQuery.Reps による絞り込みをしている箇所。
//
// rep名の絞り込みは **api/find_filter.go の findKyous（filterKyousByRepName）にだけ**置く。
// dao/reps へ降ろしてはいけない理由は、そこを通るのが利用者のクエリだけではないから:
// ReKyou / MiReKyou は参照先を解決するために target_resolution_memo.go:120 で
// **利用者のクエリをそのまま** FindKyousSequential へ渡す。ここで rep名で絞ると、
// チェックしていないrepに参照先を持つリポストが「参照先が見つからない」扱いになり、
// **語句検索に黙って当たらなくなる**（エラーも0件表示も出ず、ヒット数が減るだけ）。
//
// GkillRepositories.Reps（リポジトリの一覧そのもの）とは別物なので、
// クエリ側の受け手名でだけ引っかける。
var queryRepsFilterPattern = regexp.MustCompile(`\b(query|findQuery|parsedFindQuery|q)\.Reps\b`)

func TestNoRepNameFilterInDaoReps(t *testing.T) {
	// SQL を組み立てる層（dao/sqlite3impl）も走査する。
	// rep名の絞り込みを「速そうだから」と降ろすとしたら行き先はここで、
	// いま dao/sqlite3impl には query.Reps が1件も無い ＝ 規約が守られている状態。
	// 走査対象から外れていると、降ろされた瞬間を誰も見張っていないことになる。
	scanDirs := []string{
		filepath.Join(sourceScanGkillRoot, "dao", "reps"),
		filepath.Join(sourceScanGkillRoot, "dao", "sqlite3impl"),
	}
	violations := []string{}
	scanned := 0
	for _, scanDir := range scanDirs {
		err := filepath.WalkDir(scanDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			scanned++
			for i, line := range strings.Split(string(content), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") {
					continue // 説明コメントは対象外（この規約自体を説明した行が引っかかるため）
				}
				if queryRepsFilterPattern.MatchString(line) {
					violations = append(violations,
						fmt.Sprintf("%s:%d: %s", filepath.ToSlash(path), i+1, trimmed))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s の走査に失敗した: %v", scanDir, err)
		}
	}
	if scanned < 140 {
		t.Fatalf("dao/reps と dao/sqlite3impl のGoファイルが %d 件しか見つからない。rootの指定(%s)が間違っている可能性がある", scanned, sourceScanGkillRoot)
	}
	if len(violations) != 0 {
		t.Fatalf("dao 層で FindQuery.Reps による絞り込みをしている。"+
			"rep名の絞り込みは api/find_filter.go の filterKyousByRepName にだけ置くこと"+
			"（ReKyou/MiReKyou のワード委譲が利用者のクエリをそのまま渡すため、"+
			"ここで絞るとチェックしていないrepに参照先を持つリポストが語句検索に当たらなくなる）:\n%s",
			strings.Join(violations, "\n"))
	}
}

// commit_tx でキャッシュへ書き戻す直前に実rep名を入れ忘れている型。
//
// 一時リポジトリの Get*ByTXID は `? AS REP_NAME` に **temp rep の合成名**（"KMEMO_TEMP" 等）を
// バインドして返す。それをそのままキャッシュへ write-through すると、キャッシュ表の REP_NAME が
// 合成名になり、rep名での絞り込み（filterKyousByRepName）から漏れて
// **確定したばかりの記録が一覧から消える**。
//
// 13型すべてが同じ2行の組（`x.RepName = repName` → `WriteThroughXxxCache`）で書かれている
// コピペ形なので、1型だけ抜けても他の12型のテストは緑のまま通る。順序ごと機械で固定する。
var (
	commitTxWriteThroughPattern = regexp.MustCompile(`WriteThrough\w+Cache\(r\.Context\(\), (\w+)\)`)
	// 期待される直前の行。IDF だけは「実DBへ永続化する TargetRepName の復元」が別にあるが、
	// キャッシュ用の代入はこの形で他の12型と揃っている
	commitTxRepNameAssign = "%s.RepName = repName"
	// この数を下回ったら正規表現がずれている
	commitTxExpectedTypes = 13
)

func TestCommitTxSetsRealRepNameBeforeWriteThrough(t *testing.T) {
	path := filepath.Join(sourceScanGkillRoot, "api", "gkill_server_api", "handle_commit_tx.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s を読めない: %v", path, err)
	}
	lines := strings.Split(string(content), "\n")

	found := 0
	violations := []string{}
	for i, line := range lines {
		m := commitTxWriteThroughPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		found++
		want := fmt.Sprintf(commitTxRepNameAssign, m[1])
		// 直前の実コード行（空行とコメントは飛ばす）
		prev := ""
		prevLine := 0
		for j := i - 1; j >= 0; j-- {
			candidate := strings.TrimSpace(lines[j])
			if candidate == "" || strings.HasPrefix(candidate, "//") {
				continue
			}
			prev = candidate
			prevLine = j + 1
			break
		}
		if prev != want {
			violations = append(violations, fmt.Sprintf(
				"handle_commit_tx.go:%d: %q の直前(%d行目)が %q ではなく %q",
				i+1, strings.TrimSpace(line), prevLine, want, prev))
		}
	}
	if found < commitTxExpectedTypes {
		t.Fatalf("write-through が %d 件しか見つからない（%d 件のはず）。正規表現がずれている可能性がある",
			found, commitTxExpectedTypes)
	}
	if len(violations) != 0 {
		t.Fatalf("commit_tx でキャッシュへ書き戻す前に実rep名を入れていない型がある。"+
			"一時リポジトリの合成rep名がキャッシュへ入ると、確定した記録が rep絞り込みから漏れる:\n%s",
			strings.Join(violations, "\n"))
	}
}

// IDF だけは「キャッシュ用の rep名」とは別に、**実DBへ永続化される** TARGET_REP_NAME の
// 復元が要る。leaf の AddIDFKyouInfo が idfKyou.RepName を TARGET_REP_NAME 列として書くので、
// temp rep の合成名 "IDF_TEMP" のまま渡すと **ファイルの所在が実データごと壊れ、
// UpdateCache でも直らない**（キャッシュではないため）。この範囲で唯一の不可逆な失敗モード。
func TestCommitTxRestoresIDFTargetRepNameBeforeRealWrite(t *testing.T) {
	path := filepath.Join(sourceScanGkillRoot, "api", "gkill_server_api", "handle_commit_tx.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s を読めない: %v", path, err)
	}
	lines := strings.Split(string(content), "\n")

	addIndex := -1
	for i, line := range lines {
		if strings.Contains(line, "WriteIDFKyouRep.AddIDFKyouInfo(r.Context(), idfKyou)") {
			addIndex = i
			break
		}
	}
	if addIndex < 0 {
		t.Fatal("handle_commit_tx.go に WriteIDFKyouRep.AddIDFKyouInfo の呼び出しが無い。実装が変わった可能性がある")
	}

	// 直前の実コード行が TargetRepName の復元であること
	for j := addIndex - 1; j >= 0; j-- {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if trimmed != "idfKyou.RepName = idfKyou.TargetRepName" {
			t.Fatalf("handle_commit_tx.go:%d: AddIDFKyouInfo の直前が "+
				"%q ではなく %q。temp rep の合成名のまま書くと TARGET_REP_NAME が実DBへ入り、"+
				"ファイルの所在が壊れる（UpdateCache でも直らない）",
				j+1, "idfKyou.RepName = idfKyou.TargetRepName", trimmed)
		}
		break
	}
}
