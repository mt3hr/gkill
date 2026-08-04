package reps

import (
	"context"
	sqllib "database/sql"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
)

// stubGitCommitLogRep はネスト並列回帰テスト用の空実装
type stubGitCommitLogRep struct{}

func (s *stubGitCommitLogRep) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return map[string][]Kyou{}, nil
}

func (s *stubGitCommitLogRep) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return nil, nil
}

func (s *stubGitCommitLogRep) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return nil, nil
}

func (s *stubGitCommitLogRep) GetPath(ctx context.Context, id string) (string, error) {
	return "", nil
}

func (s *stubGitCommitLogRep) UpdateCache(ctx context.Context) error { return nil }

func (s *stubGitCommitLogRep) LastUpdateCacheChanged() bool { return false }

func (s *stubGitCommitLogRep) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	return nil, nil
}

func (s *stubGitCommitLogRep) GetRepName(ctx context.Context) (string, error) {
	return "stub_git_rep", nil
}

func (s *stubGitCommitLogRep) Close(ctx context.Context) error { return nil }

func (s *stubGitCommitLogRep) FindGitCommitLog(ctx context.Context, query *find.FindQuery) ([]GitCommitLog, error) {
	return []GitCommitLog{}, nil
}

func (s *stubGitCommitLogRep) FindGitCommitLogByIDs(ctx context.Context, ids []string) ([]GitCommitLog, error) {
	return []GitCommitLog{}, nil
}

func (s *stubGitCommitLogRep) GetGitCommitLog(ctx context.Context, id string, updateTime *time.Time) (*GitCommitLog, error) {
	return nil, nil
}

func (s *stubGitCommitLogRep) UnWrapTyped() ([]GitCommitLogRepository, error) {
	return []GitCommitLogRepository{s}, nil
}

func (s *stubGitCommitLogRep) UnWrap() ([]Repository, error) { return nil, nil }

// キャッシュビルド中フォールバックがthreads.Goスロットを保持したまま
// 集約リポジトリの並列FindKyousをネスト呼び出しし、プールが枯渇して
// 恒久ハングしていたバグの回帰テスト。
// 修正後は逐次版(FindKyousSequential等)へ逃がすため、プールに空きが
// 1つも残っていなくても完了する。
func TestGitCommitLogCachedBuildingFallbackDoesNotExhaustPool(t *testing.T) {
	ctx := context.Background()

	db, err := sqllib.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	aggregated := GitCommitLogRepositories{&stubGitCommitLogRep{}, &stubGitCommitLogRep{}}
	cachedRep, err := NewGitRepCachedSQLite3Impl(ctx, aggregated, db, nil, "GIT_COMMIT_LOG_NESTED_POOL_TEST")
	if err != nil {
		t.Fatalf("NewGitRepCachedSQLite3Impl() error: %v", err)
	}
	impl, ok := cachedRep.(*gitCommitLogRepositoryCachedSQLite3Impl)
	if !ok {
		t.Fatalf("unexpected impl type %T", cachedRep)
	}
	impl.isCacheBuilding.Store(true)
	t.Cleanup(func() { impl.isCacheBuilding.Store(false) })

	// スレッドプールを満杯にする（タイムアウト付きAcquireで空きが無くなるまで取得）
	releases := []func(){}
	t.Cleanup(func() {
		for _, release := range releases {
			release()
		}
	})
	for {
		acquireCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		release, err := threads.Acquire(acquireCtx)
		cancel()
		if err != nil {
			break // 満杯になった
		}
		releases = append(releases, release)
	}
	if len(releases) == 0 {
		t.Fatal("could not acquire any pool slot")
	}

	// 1スロットだけ空け、そのスロットを保持する閉包の中からフォールバック検索を呼ぶ。
	// 修正前はここで集約の並列FindKyousが内側スロットを永久に待ちハングしていた
	releases[len(releases)-1]()
	releases = releases[:len(releases)-1]

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg := &sync.WaitGroup{}
		err := threads.Go(ctx, wg, func() {
			if _, err := impl.FindKyous(ctx, &find.FindQuery{}); err != nil {
				t.Errorf("FindKyous() error: %v", err)
			}
			if _, err := impl.FindGitCommitLog(ctx, &find.FindQuery{}); err != nil {
				t.Errorf("FindGitCommitLog() error: %v", err)
			}
			if _, err := impl.GetGitCommitLog(ctx, "not_exist_id", nil); err != nil {
				t.Errorf("GetGitCommitLog() error: %v", err)
			}
		})
		if err != nil {
			t.Errorf("threads.Go() error: %v", err)
		}
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("deadlock: キャッシュビルド中フォールバックがスレッドプール枯渇でハングした")
	}
}

// isCacheBuildingの読み書きが検索と並行しても-raceで検出されないことの回帰テスト
// (かつては素のboolで無同期に読み書きされていた)
func TestGitCommitLogCachedIsCacheBuildingNoDataRace(t *testing.T) {
	ctx := context.Background()

	db, err := sqllib.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	aggregated := GitCommitLogRepositories{&stubGitCommitLogRep{}}
	cachedRep, err := NewGitRepCachedSQLite3Impl(ctx, aggregated, db, nil, "GIT_COMMIT_LOG_RACE_TEST")
	if err != nil {
		t.Fatalf("NewGitRepCachedSQLite3Impl() error: %v", err)
	}
	impl := cachedRep.(*gitCommitLogRepositoryCachedSQLite3Impl)

	stop := make(chan struct{})
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for {
			select {
			case <-stop:
				return
			default:
				impl.isCacheBuilding.Store(true)
				impl.isCacheBuilding.Store(false)
			}
		}
	}()
	for range 200 {
		if _, err := impl.FindKyous(ctx, &find.FindQuery{}); err != nil {
			t.Fatalf("FindKyous() error: %v", err)
		}
	}
	close(stop)
	<-writerDone
}
