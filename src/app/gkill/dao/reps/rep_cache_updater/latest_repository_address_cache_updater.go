package rep_cache_updater

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
)

type latestRepositoryAddressCacheUpdater struct {
	repository      CacheUpdatable
	gkillRepository *reps.GkillRepositories

	enableUpdateRepsCache                 bool
	enableUpdateLatestDataRepositoryCache bool

	cancelPreFunc context.CancelFunc // 一回前で実行されたコンテキスト。キャンセル用

	updateCycleDucation time.Duration
	ticker              *time.Ticker
	isUpdateNextTick    bool
	isClosed            bool

	m sync.Mutex
}

func NewLatestRepositoryAddressCacheUpdater(rep CacheUpdatable, gkillRepoisitory *reps.GkillRepositories, enableUpdateRepsCache bool, enableUpdateLatestDataRepositoryCache bool) CacheUpdatable {
	updateCycleDucation := gkill_options.CacheUpdateCycle
	latestRepositoryAddressCacheUpdater := &latestRepositoryAddressCacheUpdater{
		repository:      rep,
		gkillRepository: gkillRepoisitory,

		enableUpdateRepsCache:                 enableUpdateRepsCache,
		enableUpdateLatestDataRepositoryCache: enableUpdateLatestDataRepositoryCache,

		cancelPreFunc: context.CancelFunc(func() {}),

		updateCycleDucation: updateCycleDucation,
		ticker:              time.NewTicker(updateCycleDucation),
		isClosed:            false,

		m: sync.Mutex{},
	}

	go func() {
		for !latestRepositoryAddressCacheUpdater.isClosed {
			<-latestRepositoryAddressCacheUpdater.ticker.C
			if latestRepositoryAddressCacheUpdater.isUpdateNextTick {
				latestRepositoryAddressCacheUpdater.UpdateCacheImpl(context.Background())
			}
			latestRepositoryAddressCacheUpdater.isUpdateNextTick = false
		}
	}()

	return latestRepositoryAddressCacheUpdater
}

func (l *latestRepositoryAddressCacheUpdater) UpdateCache(ctx context.Context) error {
	l.isUpdateNextTick = true
	return nil
}

func (l *latestRepositoryAddressCacheUpdater) GetRepName(ctx context.Context) (string, error) {
	return l.repository.GetRepName(ctx)
}

func (l *latestRepositoryAddressCacheUpdater) GetPath(ctx context.Context, id string) (string, error) {
	return l.repository.GetPath(ctx, id)
}

func (l *latestRepositoryAddressCacheUpdater) Close() error {
	l.ticker.Stop()
	return nil
}

func (l *latestRepositoryAddressCacheUpdater) UpdateCacheImpl(ctx context.Context) {
	func() {
		l.m.Lock()
		defer l.m.Unlock()

		var cancelFunc context.CancelFunc

		// 一個前でUpdateCacheしてるやつをキャンセルする
		l.cancelPreFunc()
		ctx, cancelFunc = context.WithCancel(ctx)
		l.cancelPreFunc = cancelFunc
	}()
	done := threads.AllocateThread()
	go func() {
		defer done()
		if l.enableUpdateRepsCache {
			err := l.repository.UpdateCache(ctx)
			if err != nil {
				repName, _ := l.repository.GetRepName(ctx)
				err = fmt.Errorf("error at update rep. repname = %s: %w", repName, err)
				gkill_log.Debug.Print(err)
				return
			}
		}

		if l.enableUpdateLatestDataRepositoryCache {
			err := l.gkillRepository.UpdateCache(ctx)
			if err != nil {
				repName, _ := l.repository.GetRepName(ctx)
				err = fmt.Errorf("error at update latest repositoryh address dao. repname = %s: %w", repName, err)
				gkill_log.Debug.Print(err)
				return
			}
		}
	}()
}
