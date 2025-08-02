package rep_cache_updater

import (
	"context"
	"fmt"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
)

var (
	// 全体で1つだけ起動されるように考慮
	updateThreadPool = make(chan interface{}, 1)
)

func init() {
	updateThreadPool <- struct{}{}
}

type latestRepositoryAddressCacheUpdater struct {
	repository      CacheUpdatable
	gkillRepository *reps.GkillRepositories

	enableUpdateRepsCache                 bool
	enableUpdateLatestDataRepositoryCache bool

	cancelPreFunc context.CancelFunc // 一回前で実行されたコンテキスト。キャンセル用

	m sync.Mutex
}

func NewLatestRepositoryAddressCacheUpdater(rep CacheUpdatable, gkillRepoisitory *reps.GkillRepositories, enableUpdateRepsCache bool, enableUpdateLatestDataRepositoryCache bool) CacheUpdatable {
	return &latestRepositoryAddressCacheUpdater{
		repository:      rep,
		gkillRepository: gkillRepoisitory,

		enableUpdateRepsCache:                 enableUpdateRepsCache,
		enableUpdateLatestDataRepositoryCache: enableUpdateLatestDataRepositoryCache,

		cancelPreFunc: context.CancelFunc(func() {}),

		m: sync.Mutex{},
	}
}

func (l *latestRepositoryAddressCacheUpdater) UpdateCache(ctx context.Context) error {
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
		select {
		case <-ctx.Done():
			return
		default:
		}

		<-updateThreadPool
		if l.enableUpdateLatestDataRepositoryCache {
			defer func() { updateThreadPool <- struct{}{} }()
			err := l.gkillRepository.UpdateCache(context.Background())
			if err != nil {
				repName, _ := l.repository.GetRepName(context.Background())
				err = fmt.Errorf("error at update latest repositoryh address dao. repname = %s: %w", repName, err)
				gkill_log.Debug.Print(err)
				return
			}
		}
	}()
	return nil
}

func (l *latestRepositoryAddressCacheUpdater) GetRepName(ctx context.Context) (string, error) {
	return l.repository.GetRepName(ctx)
}

func (l *latestRepositoryAddressCacheUpdater) GetPath(ctx context.Context, id string) (string, error) {
	return l.repository.GetPath(ctx, id)
}
