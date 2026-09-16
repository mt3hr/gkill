package reps

import (
	"context"
	sqllib "database/sql"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// missCountingGitRep は下層への問い合わせ回数を数える stub。
// 「構築済みキャッシュで外れた ID を引いても下層へ落ちない」ことを、回数の不変で確かめる。
type missCountingGitRep struct {
	dupGitRep
	getGitCommitLogCalls  atomic.Int32
	getKyouCalls          atomic.Int32
	getKyouHistoriesCalls atomic.Int32
}

func (s *missCountingGitRep) GetGitCommitLog(ctx context.Context, id string, updateTime *time.Time) (*GitCommitLog, error) {
	s.getGitCommitLogCalls.Add(1)
	for _, commit := range s.commits {
		if commit.ID == id {
			found := commit
			return &found, nil
		}
	}
	return nil, nil
}

func (s *missCountingGitRep) kyouOf(id string) *Kyou {
	for _, commit := range s.commits {
		if commit.ID == id {
			return &Kyou{
				ID: commit.ID, RepName: commit.RepName, DataType: "git_commit_log",
				RelatedTime: commit.RelatedTime, CreateTime: commit.CreateTime, UpdateTime: commit.UpdateTime,
			}
		}
	}
	return nil
}

func (s *missCountingGitRep) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	s.getKyouCalls.Add(1)
	return s.kyouOf(id), nil
}

func (s *missCountingGitRep) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	s.getKyouHistoriesCalls.Add(1)
	kyou := s.kyouOf(id)
	if kyou == nil {
		return nil, nil
	}
	return []Kyou{*kyou}, nil
}

// 構築済みのキャッシュで ID が外れても、下層の生リポジトリへ落ちないこと。
//
// 2026-09-16 まで「SQL で 0 件 → 下層へフォールバック」が無条件で、archived プラグインのコミット
// （native のキャッシュに無い ID）を rykv の行ごとに引かれるたびに 18 リポジトリの git log 全走査が走り、
// 1 行約 1 秒×多コアでサーバが飽和していた（ADR-0221）。
// フォールバックが許されるのは「まだ構築していない」「バックグラウンド構築中」のときだけ。
func TestGitCommitLogCachedBuiltCacheDoesNotFallBackOnMiss(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "git_cache_miss.db")
	db, err := sqllib.Open("sqlite", dbPath+"?_txlock=immediate&_pragma=busy_timeout(6000)")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	stub := &missCountingGitRep{dupGitRep: dupGitRep{commits: []GitCommitLog{
		newTestGitCommit("commit-01"),
		newTestGitCommit("commit-02"),
	}}}
	aggregated := GitCommitLogRepositories{stub}
	cachedRep, err := NewGitRepCachedSQLite3Impl(ctx, aggregated, db, nil, "GIT_COMMIT_LOG_MISS_TEST")
	if err != nil {
		t.Fatalf("NewGitRepCachedSQLite3Impl() error: %v", err)
	}
	impl := cachedRep.(*gitCommitLogRepositoryCachedSQLite3Impl)

	// 構築前: キャッシュはまだ実リポジトリを写していないので、外れた ID は下層へ聞きに行く
	if _, err := impl.GetGitCommitLog(ctx, "missing", nil); err != nil {
		t.Fatalf("GetGitCommitLog() before build error: %v", err)
	}
	if got := stub.getGitCommitLogCalls.Load(); got != 1 {
		t.Fatalf("構築前は下層へフォールバックするはず: GetGitCommitLog calls = %d, want 1", got)
	}

	if err := impl.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache() error: %v", err)
	}

	// 構築後: キャッシュにある ID はキャッシュから返る
	found, err := impl.GetGitCommitLog(ctx, "commit-01", nil)
	if err != nil {
		t.Fatalf("GetGitCommitLog(commit-01) error: %v", err)
	}
	if found == nil || found.ID != "commit-01" {
		t.Fatalf("GetGitCommitLog(commit-01) = %v, want the cached commit", found)
	}

	// 構築後: 外れた ID は「無い」で即返り、下層へは聞かない
	missing, err := impl.GetGitCommitLog(ctx, "missing", nil)
	if err != nil {
		t.Fatalf("GetGitCommitLog(missing) error: %v", err)
	}
	if missing != nil {
		t.Errorf("GetGitCommitLog(missing) = %v, want nil", missing)
	}
	kyou, err := impl.GetKyou(ctx, "missing", nil)
	if err != nil {
		t.Fatalf("GetKyou(missing) error: %v", err)
	}
	if kyou != nil {
		t.Errorf("GetKyou(missing) = %v, want nil", kyou)
	}
	histories, err := impl.GetKyouHistories(ctx, "missing")
	if err != nil {
		t.Fatalf("GetKyouHistories(missing) error: %v", err)
	}
	if len(histories) != 0 {
		t.Errorf("GetKyouHistories(missing) len = %d, want 0", len(histories))
	}
	if got := stub.getGitCommitLogCalls.Load(); got != 1 {
		t.Errorf("構築済みなのに GetGitCommitLog が下層へ落ちた: calls = %d, want 1", got)
	}
	if got := stub.getKyouCalls.Load(); got != 0 {
		t.Errorf("構築済みなのに GetKyou が下層へ落ちた: calls = %d, want 0", got)
	}
	if got := stub.getKyouHistoriesCalls.Load(); got != 0 {
		t.Errorf("構築済みなのに GetKyouHistories が下層へ落ちた: calls = %d, want 0", got)
	}

	// バックグラウンド構築中は従来どおり下層へ落ちる
	// （TestGitCommitLogCachedBuildingFallbackReturnsUnderlyingData と同じ約束）
	impl.isCacheBuilding.Store(true)
	t.Cleanup(func() { impl.isCacheBuilding.Store(false) })
	if _, err := impl.GetGitCommitLog(ctx, "missing", nil); err != nil {
		t.Fatalf("GetGitCommitLog() while building error: %v", err)
	}
	if got := stub.getGitCommitLogCalls.Load(); got != 2 {
		t.Errorf("構築中は下層へフォールバックするはず: GetGitCommitLog calls = %d, want 2", got)
	}
}
