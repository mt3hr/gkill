package reps

import (
	"context"
	"fmt"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

type notificationRepositorySQLite3ImplLocalCached struct {
	userID               string
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          NotificationRepository
	localCachedRep       NotificationRepository
	m                    sync.RWMutex

	fullConnect bool
	// cacheChange は元DBファイルが前回のキャッシュ再構築から変わったかを見ます。
	// 上位のキャッシュrepはこれが false のときフルリビルドを飛ばします。
	cacheChange *dbFileChangeDetector

	// skipRebuild はローカルキャッシュを更新できなかった回だけ立ちます。
	// 古いコピーのまま上位に再構築させると、そのあと基準が進んで
	// 新しい内容が二度と取り込まれなくなるため、その回だけ再構築を見送らせます。
	skipRebuild bool
}

func NewNotificationRepositorySQLite3ImplLocalCached(ctx context.Context, userID string, filename string, fullConnect bool) (NotificationRepository, error) {
	localCacheDBFileName, err := localRepCacheDBFileName(userID, filename)
	if err != nil {
		return nil, err
	}
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return nil, err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(filename)
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
	if updateCache {
		originalDBFile, err := os.Open(filename)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", filename, err)
			return nil, err
		}
		defer func() {
			err := originalDBFile.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
		cacheDBFile, err := os.Create(localCacheDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName, err)
			return nil, err
		}
		defer func() {
			err := cacheDBFile.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
		_, err = io.Copy(cacheDBFile, originalDBFile)
		if err != nil {
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", filename, localCacheDBFileName, err)
			return nil, err
		}
		os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
	}

	originalRep, err := NewNotificationRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new notification rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewNotificationRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new notification rep: %w", err)
		return nil, err
	}

	cachedRep := &notificationRepositorySQLite3ImplLocalCached{
		userID:               userID,
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		cacheChange: &dbFileChangeDetector{},

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}
func (n *notificationRepositorySQLite3ImplLocalCached) FindNotifications(ctx context.Context, query *find.FindQuery) ([]Notification, error) {
	if query.UpdateCache {
		if err := n.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	n.m.RLock()
	defer n.m.RUnlock()
	return n.localCachedRep.FindNotifications(ctx, query)
}

func (n *notificationRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	n.m.Lock()
	defer n.m.Unlock()
	err := n.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	err = n.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	return nil
}

func (n *notificationRepositorySQLite3ImplLocalCached) GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	return n.localCachedRep.GetNotification(ctx, id, updateTime)
}

func (n *notificationRepositorySQLite3ImplLocalCached) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	return n.localCachedRep.GetNotificationsByTargetID(ctx, target_id)
}

func (n *notificationRepositorySQLite3ImplLocalCached) GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	return n.localCachedRep.GetNotificationsBetweenNotificationTime(ctx, startTime, endTime)
}

func (n *notificationRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	n.m.Lock()
	defer n.m.Unlock()

	// 上位のキャッシュrepへ「元DBが変わったか」を伝えるための観測。
	// 基準が進むのは再構築が成功したときのCommitCacheRebuildだけなので、
	// 途中で失敗した回のぶんは次回も再構築される。
	n.cacheChange.refresh(n.originalDBFileName)
	n.skipRebuild = false

	// 元DBがローカルキャッシュと同じなら、閉じる・消す・コピーし直す・開き直すを全部飛ばす。
	// この判定を os.Remove のあとに置くと、消した直後なので必ず「要コピー」になり、
	// 変更のないrepまで LastUpdateCacheChanged() が true を返して
	// 上位のキャッシュrepが毎回フルリビルドする。
	if !localRepCacheNeedsCopy(n.originalDBFileName, n.localCacheDBFileName) {
		return nil
	}

	err := n.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache: %w", err)
		return err
	}

	// ローカルキャッシュファイルを消せなくても致命的なエラーにはしない。
	// 別のRepositoryや別プロセスが同じファイルを開いていると
	// Windowsでは削除が共有違反で失敗する。
	// ここで error を返すと GetRepositories 全体が失敗し、
	// リポジトリがキャッシュに登録されないままリクエスト毎に再ロードされ続けるため、
	// この回の再取得だけ諦めて、閉じたハンドルを開き直して継続する。
	if err = os.Remove(n.localCacheDBFileName); err != nil && !os.IsNotExist(err) {
		slog.Log(ctx, gkill_log.Warn, "skip local rep cache refresh", "file", fmt.Sprintf("%q", n.localCacheDBFileName), "error", fmt.Sprintf("%q", err))
		n.skipRebuild = true
		reopenedRep, reopenErr := NewNotificationRepositorySQLite3Impl(ctx, n.localCacheDBFileName, n.fullConnect)
		if reopenErr != nil {
			return fmt.Errorf("error at reopen local cached rep %s: %w", n.localCacheDBFileName, reopenErr)
		}
		n.localCachedRep = reopenedRep
		return nil
	}

	localCacheDBFileName, err := localRepCacheDBFileName(n.userID, n.originalDBFileName)
	if err != nil {
		return err
	}
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	// ここへ来るのは冒頭の判定でコピーが要ると分かったときだけ。
	if err := copyLocalRepCacheDB(n.originalDBFileName, localCacheDBFileName); err != nil {
		return err
	}

	newLocalCachedRep, err := NewNotificationRepositorySQLite3Impl(ctx, localCacheDBFileName, n.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new notification rep: %w", err)
		return err
	}
	n.localCachedRep = newLocalCachedRep
	return nil
}

func (n *notificationRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	return n.originalRep.GetPath(ctx, id)
}

func (n *notificationRepositorySQLite3ImplLocalCached) LastUpdateCacheChanged() bool {
	// ローカルキャッシュを更新できなかった回は、古いコピーから作り直させない。
	if n.skipRebuild {
		return false
	}
	return n.cacheChange.lastChanged()
}

// CommitCacheRebuild は上位のキャッシュrepが再構築に成功したときに呼ばれます。
func (n *notificationRepositorySQLite3ImplLocalCached) CommitCacheRebuild() {
	n.cacheChange.commit()
}

func (n *notificationRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return n.originalRep.GetRepName(ctx)
}

func (n *notificationRepositorySQLite3ImplLocalCached) GetNotificationHistories(ctx context.Context, id string) ([]Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	return n.localCachedRep.GetNotificationHistories(ctx, id)
}

func (n *notificationRepositorySQLite3ImplLocalCached) AddNotificationInfo(ctx context.Context, notification Notification) error {
	err := func() error {
		n.m.Lock()
		defer n.m.Unlock()
		err := n.originalRep.AddNotificationInfo(ctx, notification)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}
	return n.UpdateCache(ctx)
}

func (n *notificationRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]NotificationRepository, error) {
	return []NotificationRepository{n.originalRep}, nil
}

func (n *notificationRepositorySQLite3ImplLocalCached) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	addrs, err := n.localCachedRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	repName, err := n.GetRepName(ctx)
	if err != nil {
		return nil, err
	}
	for idx := range addrs {
		addrs[idx].LatestDataRepositoryName = repName
	}
	return addrs, nil
}
