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

// stubMiReKyouRep はネスト並列回帰テスト用の空実装
type stubMiReKyouRep struct{}

func (s *stubMiReKyouRep) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return map[string][]Kyou{}, nil
}

func (s *stubMiReKyouRep) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return nil, nil
}

func (s *stubMiReKyouRep) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return nil, nil
}

func (s *stubMiReKyouRep) GetPath(ctx context.Context, id string) (string, error) {
	return "", nil
}

func (s *stubMiReKyouRep) UpdateCache(ctx context.Context) error { return nil }

func (s *stubMiReKyouRep) LastUpdateCacheChanged() bool { return false }

func (s *stubMiReKyouRep) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	return nil, nil
}

func (s *stubMiReKyouRep) GetRepName(ctx context.Context) (string, error) {
	return "stub_mirekyou_rep", nil
}

func (s *stubMiReKyouRep) Close(ctx context.Context) error { return nil }

func (s *stubMiReKyouRep) FindMiReKyou(ctx context.Context, query *find.FindQuery) ([]MiReKyou, error) {
	return []MiReKyou{}, nil
}

func (s *stubMiReKyouRep) GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error) {
	return nil, nil
}

func (s *stubMiReKyouRep) GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error) {
	return nil, nil
}

func (s *stubMiReKyouRep) AddMiReKyouInfo(ctx context.Context, mirekyou MiReKyou) error {
	return nil
}

func (s *stubMiReKyouRep) GetMiReKyousAllLatest(ctx context.Context) ([]MiReKyou, error) {
	return []MiReKyou{}, nil
}

func (s *stubMiReKyouRep) GetBoardNames(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (s *stubMiReKyouRep) GetRepositoriesWithoutMiReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	return nil, nil
}

func (s *stubMiReKyouRep) UnWrapTyped() ([]MiReKyouRepository, error) {
	return []MiReKyouRepository{s}, nil
}

func (s *stubMiReKyouRep) UnWrap() ([]Repository, error) { return nil, nil }

// newCachedMiReKyouForPoolTest はテスト用のキャッシュ実装を作ります。
func newCachedMiReKyouForPoolTest(t *testing.T, ctx context.Context, reps ...MiReKyouRepository) *miReKyouRepositoryCachedSQLite3Impl {
	t.Helper()

	db, err := sqllib.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	aggregated := &MiReKyouRepositories{MiReKyouRepositories: reps}
	cachedRep, err := NewMiReKyouRepositoryCachedSQLite3Impl(ctx, aggregated, nil, db, nil, "MIREKYOU_NESTED_POOL_TEST")
	if err != nil {
		t.Fatalf("NewMiReKyouRepositoryCachedSQLite3Impl() error: %v", err)
	}
	impl, ok := cachedRep.(*miReKyouRepositoryCachedSQLite3Impl)
	if !ok {
		t.Fatalf("unexpected impl type %T", cachedRep)
	}
	return impl
}

// fillThreadPoolLeavingOneSlot はスレッドプールを満杯にしたうえで1スロットだけ空けます。
func fillThreadPoolLeavingOneSlot(t *testing.T, ctx context.Context) {
	t.Helper()

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

	// 1スロットだけ空ける。呼び出し側はこのスロットを保持したまま委譲を呼ぶ
	releases[len(releases)-1]()
	releases = releases[:len(releases)-1]
}

// キャッシュ実装がthreads.Goスロットを保持したまま集約の並列メソッドを
// ネスト呼び出しし、プールが枯渇して恒久ハングしていたバグの回帰テスト。
//
// 修正後は逐次版(GetKyouSequential等)へ逃がすため、
// プールに空きが1つも残っていなくても内側のスロットを要求しない。
//
// threads.Goには枯渇時のinlineフォールバックが入っているので「返ってくること」
// だけでは修正を検知できない。内側でスロットを取ろうとしていないことを
// inlineフォールバック回数が増えないことで確認する。
// 修正前のコードではこのカウンタが増えてFAILする。
func TestMiReKyouCachedDelegationDoesNotExhaustPool(t *testing.T) {
	ctx := context.Background()

	impl := newCachedMiReKyouForPoolTest(t, ctx, &stubMiReKyouRep{}, &stubMiReKyouRep{})
	fillThreadPoolLeavingOneSlot(t, ctx)

	inlineFallbackBefore := threads.InlineFallbackCount()

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg := &sync.WaitGroup{}
		err := threads.Go(ctx, wg, func() {
			if _, err := impl.GetKyou(ctx, "not_exist_id", nil); err != nil {
				t.Errorf("GetKyou() error: %v", err)
			}
			if _, err := impl.GetKyouHistories(ctx, "not_exist_id"); err != nil {
				t.Errorf("GetKyouHistories() error: %v", err)
			}
			if _, err := impl.GetPath(ctx, "not_exist_id"); err == nil {
				// 見つからないのでエラーになるのが正しい。ハングしないことだけを見る
				t.Log("GetPath() unexpectedly succeeded")
			}
			if _, err := impl.GetMiReKyou(ctx, "not_exist_id", nil); err != nil {
				t.Errorf("GetMiReKyou() error: %v", err)
			}
			if _, err := impl.GetMiReKyouHistories(ctx, "not_exist_id"); err != nil {
				t.Errorf("GetMiReKyouHistories() error: %v", err)
			}
			if err := impl.UpdateCache(ctx); err != nil {
				t.Errorf("UpdateCache() error: %v", err)
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
		t.Fatal("deadlock: MiReKyouキャッシュ実装の委譲がスレッドプール枯渇でハングした")
	}

	if got := threads.InlineFallbackCount(); got != inlineFallbackBefore {
		t.Fatalf("inline fallback count changed (%d -> %d): "+
			"キャッシュ実装が集約の並列メソッドを呼んで内側のスロットを要求している",
			inlineFallbackBefore, got)
	}
}

// stubMiReKyouRepWithData は委譲の正当性テスト用。GetKyou等が実データを返す
type stubMiReKyouRepWithData struct {
	stubMiReKyouRep
}

func (s *stubMiReKyouRepWithData) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return &Kyou{ID: id, DataType: "mirekyou_create", RepName: "stub_mirekyou_rep_with_data"}, nil
}

func (s *stubMiReKyouRepWithData) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return []Kyou{{ID: id, DataType: "mirekyou_create", RepName: "stub_mirekyou_rep_with_data"}}, nil
}

func (s *stubMiReKyouRepWithData) GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error) {
	return &MiReKyou{ID: id, RepName: "stub_mirekyou_rep_with_data"}, nil
}

func (s *stubMiReKyouRepWithData) GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error) {
	return []MiReKyou{{ID: id, RepName: "stub_mirekyou_rep_with_data"}}, nil
}

func (s *stubMiReKyouRepWithData) UnWrapTyped() ([]MiReKyouRepository, error) {
	return []MiReKyouRepository{s}, nil
}

// 逐次版へ切り替えたことでデータを落としていないことの回帰テスト。
// 並列版と同じく下層リポジトリのデータが返る。
func TestMiReKyouCachedDelegationReturnsUnderlyingData(t *testing.T) {
	ctx := context.Background()

	impl := newCachedMiReKyouForPoolTest(t, ctx, &stubMiReKyouRepWithData{})

	kyou, err := impl.GetKyou(ctx, "target_id", nil)
	if err != nil {
		t.Fatalf("GetKyou() error: %v", err)
	}
	if kyou == nil || kyou.ID != "target_id" {
		t.Fatalf("GetKyou() = %v, want Kyou with ID target_id", kyou)
	}

	kyouHistories, err := impl.GetKyouHistories(ctx, "target_id")
	if err != nil {
		t.Fatalf("GetKyouHistories() error: %v", err)
	}
	if len(kyouHistories) != 1 {
		t.Fatalf("GetKyouHistories() returned %d items, want 1", len(kyouHistories))
	}

	mirekyou, err := impl.GetMiReKyou(ctx, "target_id", nil)
	if err != nil {
		t.Fatalf("GetMiReKyou() error: %v", err)
	}
	if mirekyou == nil || mirekyou.ID != "target_id" {
		t.Fatalf("GetMiReKyou() = %v, want MiReKyou with ID target_id", mirekyou)
	}

	mirekyouHistories, err := impl.GetMiReKyouHistories(ctx, "target_id")
	if err != nil {
		t.Fatalf("GetMiReKyouHistories() error: %v", err)
	}
	if len(mirekyouHistories) != 1 {
		t.Fatalf("GetMiReKyouHistories() returned %d items, want 1", len(mirekyouHistories))
	}
}
