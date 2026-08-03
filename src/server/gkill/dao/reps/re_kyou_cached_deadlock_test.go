package reps

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// FindKyous/FindReKyouとUpdateCacheの並行実行がデッドロックしないことの回帰テスト。
// かつてFindKyousが共有RWMutexのRLockを保持したままLatestDataRepositoryAddressDAO等を
// 再帰RLockしており、UpdateCacheのLock要求が間に割り込むと恒久デッドロックしていた
// (検索が二度と返らなくなり、再起動するまで直らない)。
func TestReKyouCachedFindKyousDoesNotDeadlockWithUpdateCache(t *testing.T) {
	ctx := context.Background()
	old := gkill_options.CacheReKyouReps
	enable := true
	gkill_options.CacheReKyouReps = &enable
	t.Cleanup(func() { gkill_options.CacheReKyouReps = old })

	repositories, _, _, _ := newGranularReKyouFixture(t)
	if err := repositories.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache() error: %v", err)
	}
	// 実際の検索経路(Repositories.findKyous)と同じく、cached実装のFindKyous/FindReKyouを直接叩く
	cachedReKyouRep := repositories.ReKyouReps.ReKyouRepositories[0]

	done := make(chan struct{})
	findersDone := make(chan struct{})
	go func() {
		defer close(done)
		wg := &sync.WaitGroup{}
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 100 {
					if _, err := cachedReKyouRep.FindKyous(ctx, makeDefaultFindQuery()); err != nil {
						t.Errorf("FindKyous() error: %v", err)
						return
					}
					if _, err := cachedReKyouRep.FindReKyou(ctx, makeDefaultFindQuery()); err != nil {
						t.Errorf("FindReKyou() error: %v", err)
						return
					}
				}
			}()
		}
		// 共有RWMutexのLockを最短周期で要求し続け、FindKyousがRLockを保持したまま
		// 再帰RLockする隙間(かつて存在したバグの窓)へ書き込み要求を必ず割り込ませる。
		// UpdateCacheやAddReKyouInfoなど、同じmutexでLockを取る書き込み全般の代役
		writerWG := &sync.WaitGroup{}
		writerWG.Add(1)
		go func() {
			defer writerWG.Done()
			for {
				select {
				case <-findersDone:
					return
				default:
				}
				repositories.CacheMemoryDBMutex.Lock()
				repositories.CacheMemoryDBMutex.Unlock() //nolint:staticcheck // 空クリティカルセクションはLock割り込み再現のため意図的
			}
		}()
		wg.Wait()
		close(findersDone)
		writerWG.Wait()
	}()

	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("deadlock: FindKyous/FindReKyouとUpdateCacheの並行実行が完了しなかった")
	}
}
