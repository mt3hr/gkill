package reps

// gitリポジトリを直接読むrep(gitCommitLogRepositoryLocalImpl)のフィルタ判定のテスト。
//
// このrepはSQLを通さずGo側で1コミットずつ判定するので、
// SQL側(GenerateFindSQLCommon)と意味論を手で揃える必要がある。
// 揃っていないと「gitのrepだけ絞り込みが効かない」という形で表に出る。

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// newTempGitCommitLogRepo は2コミットだけ入ったgitリポジトリのrepと、
// その2コミットのハッシュ(古い順)を返す。
// フィルタが見るのは Committer.When なので、そこを固定時刻にする。
func newTempGitCommitLogRepo(t *testing.T) (rep GitCommitLogRepository, firstHash string, secondHash string) {
	t.Helper()

	// go-gitが開いたファイルがWindowsで残ると t.TempDir() の自動削除が失敗するので、
	// 自前で作って後片付けはbest-effortにする
	dir, err := os.MkdirTemp("", "gkill_git_commit_log_rep_test")
	if err != nil {
		t.Fatalf("os.MkdirTemp failed: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	gitRep, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit failed: %v", err)
	}
	worktree, err := gitRep.Worktree()
	if err != nil {
		t.Fatalf("Worktree failed: %v", err)
	}

	commit := func(fileName string, when time.Time) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, fileName), []byte(fileName+"\n"), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
		if _, err := worktree.Add(fileName); err != nil {
			t.Fatalf("worktree.Add failed: %v", err)
		}
		signature := &object.Signature{Name: "test_user", Email: "test_user@example.com", When: when}
		hash, err := worktree.Commit("commit "+fileName, &git.CommitOptions{Author: signature, Committer: signature})
		if err != nil {
			t.Fatalf("worktree.Commit failed: %v", err)
		}
		return hash.String()
	}

	firstHash = commit("first.txt", gitCommitLogFirstTime())
	secondHash = commit("second.txt", gitCommitLogSecondTime())

	rep, err = NewGitRep(dir)
	if err != nil {
		t.Fatalf("NewGitRep failed: %v", err)
	}
	t.Cleanup(func() { _ = rep.Close(context.Background()) })
	return rep, firstHash, secondHash
}

// gitCommitLogFirstTime / gitCommitLogSecondTime は2コミットのコミット時刻。
func gitCommitLogFirstTime() time.Time  { return testTime() }
func gitCommitLogSecondTime() time.Time { return testTime().Add(2 * time.Hour) }

// countGitCommitLogKyous は FindKyous の戻り値のKyou総数を返す。
func countGitCommitLogKyous(kyous map[string][]Kyou) int {
	count := 0
	for _, kyousOfID := range kyous {
		count += len(kyousOfID)
	}
	return count
}

// gitのコミットに削除の概念は無いので、削除済み検索には1件も該当しない。
func TestGitCommitLogLocalDirFindKyousIsDeletedMatchesNothing(t *testing.T) {
	rep, _, _ := newTempGitCommitLogRepo(t)
	ctx := context.Background()

	kyous, err := rep.FindKyous(ctx, &find.FindQuery{IsDeleted: true})
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	if got := countGitCommitLogKyous(kyous); got != 0 {
		t.Errorf("削除済み検索にgitコミットが該当してはいけない: got %d件", got)
	}
}

// IDsの意味論をSQL側(GenerateFindSQLCommon)と揃える。
// nil=未使用(全件) / 非nil空=明示的な0件指定 / 指定あり=そのIDだけ。
// 以前は非nil空でループが回らず match=true のまま全コミットが返っていた。
func TestGitCommitLogLocalDirFindKyousIDsSemantics(t *testing.T) {
	rep, firstHash, _ := newTempGitCommitLogRepo(t)
	ctx := context.Background()

	nilIDsKyous, err := rep.FindKyous(ctx, &find.FindQuery{})
	if err != nil {
		t.Fatalf("FindKyous(IDs=nil) failed: %v", err)
	}
	if got := countGitCommitLogKyous(nilIDsKyous); got != 2 {
		t.Errorf("IDs=nil は未使用なので全コミットが返るはず: got %d件", got)
	}

	emptyIDsKyous, err := rep.FindKyous(ctx, &find.FindQuery{IDs: []string{}})
	if err != nil {
		t.Fatalf("FindKyous(IDs=[]) failed: %v", err)
	}
	if got := countGitCommitLogKyous(emptyIDsKyous); got != 0 {
		t.Errorf("IDs=[] は明示的な0件指定なので1件も返してはいけない: got %d件", got)
	}

	oneIDKyous, err := rep.FindKyous(ctx, &find.FindQuery{IDs: []string{firstHash}})
	if err != nil {
		t.Fatalf("FindKyous(IDs=[hash]) failed: %v", err)
	}
	if got := countGitCommitLogKyous(oneIDKyous); got != 1 {
		t.Fatalf("指定した1コミットだけが返るはず: got %d件", got)
	}
	if len(oneIDKyous[firstHash]) != 1 {
		t.Errorf("指定したハッシュのKyouが返っていない: got %v", oneIDKyous)
	}
}

// カレンダー範囲は両端を含む(SQL側のunixepoch >= / <= と同じ)。
// 以前は排他で、境界ちょうどのコミットが落ちていた。
func TestGitCommitLogLocalDirFindKyousCalendarRangeIncludesBothEnds(t *testing.T) {
	rep, firstHash, secondHash := newTempGitCommitLogRepo(t)
	ctx := context.Background()

	firstTime := gitCommitLogFirstTime()
	secondTime := gitCommitLogSecondTime()

	// 両端がそれぞれのコミット時刻ちょうど。2件とも含まれる
	bothKyous, err := rep.FindKyous(ctx, makeCalendarFindQuery(firstTime, secondTime))
	if err != nil {
		t.Fatalf("FindKyous(両端ちょうど) failed: %v", err)
	}
	if got := countGitCommitLogKyous(bothKyous); got != 2 {
		t.Errorf("範囲の両端は含むはず: got %d件", got)
	}

	// startだけがコミット時刻ちょうど(幅ゼロ)。古い方だけが含まれる
	firstOnlyKyous, err := rep.FindKyous(ctx, makeCalendarFindQuery(firstTime, firstTime))
	if err != nil {
		t.Fatalf("FindKyous(古い方ちょうど) failed: %v", err)
	}
	if len(firstOnlyKyous[firstHash]) != 1 || countGitCommitLogKyous(firstOnlyKyous) != 1 {
		t.Errorf("開始=終了=古い方のコミット時刻なら古い方だけが返るはず: got %v", firstOnlyKyous)
	}

	// endだけがコミット時刻ちょうど(幅ゼロ)。新しい方だけが含まれる
	secondOnlyKyous, err := rep.FindKyous(ctx, makeCalendarFindQuery(secondTime, secondTime))
	if err != nil {
		t.Fatalf("FindKyous(新しい方ちょうど) failed: %v", err)
	}
	if len(secondOnlyKyous[secondHash]) != 1 || countGitCommitLogKyous(secondOnlyKyous) != 1 {
		t.Errorf("開始=終了=新しい方のコミット時刻なら新しい方だけが返るはず: got %v", secondOnlyKyous)
	}
}

// git_commit_logのrep設定は `$HOME/Git/*` のようなglobで書かれ、
// zglobはディレクトリだけでなくファイルも返すため、
// 展開先にgitリポジトリでないエントリが混ざるのは異常ではない。
// 呼び出し側がそれを「そのrepだけスキップ」と判断できるよう、
// NewGitRepはErrNotGitRepositoryを包んだエラーを返さなければならない。
// (これを素のエラーに戻すとGetRepositories全体が失敗し、
// 対象ユーザの全APIがERR000018になって何もできなくなる)
func TestNewGitRepReturnsErrNotGitRepositoryForNonGitPath(t *testing.T) {
	dir, err := os.MkdirTemp("", "gkill_git_commit_log_not_a_repo_test")
	if err != nil {
		t.Fatalf("os.MkdirTemp failed: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	// gitリポジトリでないファイル(Git Bashのクラッシュダンプ等が実際に混ざる)
	strayFile := filepath.Join(dir, "bash.exe.stackdump")
	if err := os.WriteFile(strayFile, []byte("stack trace\n"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// gitリポジトリでないディレクトリ
	strayDir := filepath.Join(dir, "not_a_git_repository")
	if err := os.MkdirAll(strayDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	for _, path := range []string{strayFile, strayDir, filepath.Join(dir, "not_exist")} {
		rep, err := NewGitRep(path)
		if err == nil {
			_ = rep.Close(context.Background())
			t.Errorf("gitリポジトリでない %s でエラーにならなかった", path)
			continue
		}
		if !errors.Is(err, ErrNotGitRepository) {
			t.Errorf("gitリポジトリでない %s のエラーはErrNotGitRepositoryを包むはず: got %v", path, err)
		}
	}
}

// 正しいgitリポジトリではErrNotGitRepositoryにならないこと。
// (常にErrNotGitRepository扱いにしてしまうと、全部スキップされて
// gitのコミットログが1件も出なくなるのに気づけない)
func TestNewGitRepSuccessForGitRepository(t *testing.T) {
	rep, _, _ := newTempGitCommitLogRepo(t)
	if rep == nil {
		t.Fatal("gitリポジトリなのにrepが生成されなかった")
	}
}
