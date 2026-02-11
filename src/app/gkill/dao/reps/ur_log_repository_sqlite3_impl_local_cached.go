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

type urlogRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          URLogRepository
	localCachedRep       URLogRepository
	m                    sync.RWMutex

	fullConnect bool
}

func NewURLogRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (URLogRepository, error) {
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

	originalRep, err := NewURLogRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new urlog rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewURLogRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new urlog rep: %w", err)
		return nil, err
	}

	cachedRep := &urlogRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}

func (u *urlogRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.FindKyous(ctx, query)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.GetKyouHistories(ctx, id)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.originalRep.GetPath(ctx, id)
}

func (u *urlogRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	u.m.Lock()
	defer u.m.Unlock()

	err := u.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache %s", err)
		return err
	}

	err = os.Remove(u.localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at remove %s: %w", u.localCacheDBFileName, err)
		return err
	}

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.ReplaceAll(u.originalDBFileName, ":", ""))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(u.originalDBFileName)
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
	if updateCache {
		originalDBFile, err := os.Open(u.originalDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", u.originalDBFileName, err)
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
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", u.originalDBFileName, localCacheDBFileName, err)
			return err
		}
		os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
	}

	newLocalCachedRep, err := NewURLogRepositorySQLite3Impl(ctx, localCacheDBFileName, u.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new urlog rep: %w", err)
		return err
	}
	u.localCachedRep = newLocalCachedRep
	return nil
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return u.originalRep.GetRepName(ctx)
}

func (u *urlogRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	u.m.Lock()
	defer u.m.Unlock()
	err := u.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	err = u.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	return nil
}

func (u *urlogRepositorySQLite3ImplLocalCached) FindURLog(ctx context.Context, query *find.FindQuery) ([]*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.FindURLog(ctx, query)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.GetURLog(ctx, id, updateTime)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetURLogHistories(ctx context.Context, id string) ([]*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.GetURLogHistories(ctx, id)
}

func (u *urlogRepositorySQLite3ImplLocalCached) AddURLogInfo(ctx context.Context, urlog *URLog) error {
	u.m.Lock()
	defer u.m.Unlock()
	err := u.originalRep.AddURLogInfo(ctx, urlog)
	if err != nil {
		return err
	}
	return u.UpdateCache(ctx)
}

func (u *urlogRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]URLogRepository, error) {
	return []URLogRepository{u.originalRep}, nil
}

func (u *urlogRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{u.originalRep}, nil
}
