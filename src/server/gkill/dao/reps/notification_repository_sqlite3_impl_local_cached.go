package reps

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type notificationRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          NotificationRepository
	localCachedRep       NotificationRepository
	m                    sync.RWMutex

	fullConnect bool
}

func NewNotificationRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (NotificationRepository, error) {
	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.ReplaceAll(filename, ":", ""))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err := os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
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
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}
func (n *notificationRepositorySQLite3ImplLocalCached) FindNotifications(ctx context.Context, query *find.FindQuery) ([]Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	return n.localCachedRep.FindNotifications(ctx, query)
}

func (n *notificationRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	n.m.Lock()
	defer n.m.Unlock()
	err := n.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	err = n.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
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

	err := n.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache %s", err)
		return err
	}

	err = os.Remove(n.localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at remove %s: %w", n.localCacheDBFileName, err)
		return err
	}

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.ReplaceAll(n.originalDBFileName, ":", ""))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(n.originalDBFileName)
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
	if updateCache {
		originalDBFile, err := os.Open(n.originalDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", n.originalDBFileName, err)
			return err
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
			return err
		}
		defer func() {
			err := cacheDBFile.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
		_, err = io.Copy(cacheDBFile, originalDBFile)
		if err != nil {
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", n.originalDBFileName, localCacheDBFileName, err)
			return err
		}
		os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
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
