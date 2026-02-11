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

type kmemoRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          KmemoRepository
	localCachedRep       KmemoRepository
	m                    sync.RWMutex

	fullConnect bool
}

func NewKmemoRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (KmemoRepository, error) {
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

	originalRep, err := NewKmemoRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new kmemo rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewKmemoRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new kmemo rep: %w", err)
		return nil, err
	}

	cachedRep := &kmemoRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}

func (k *kmemoRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.FindKyous(ctx, query)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.GetKyouHistories(ctx, id)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.originalRep.GetPath(ctx, id)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	k.m.Lock()
	defer k.m.Unlock()

	err := k.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache %s", err)
		return err
	}

	err = os.Remove(k.localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at remove %s: %w", k.localCacheDBFileName, err)
		return err
	}

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.ReplaceAll(k.originalDBFileName, ":", ""))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(k.originalDBFileName)
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
	if updateCache {
		originalDBFile, err := os.Open(k.originalDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", k.originalDBFileName, err)
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
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", k.originalDBFileName, localCacheDBFileName, err)
			return err
		}
		os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
	}

	newLocalCachedRep, err := NewKmemoRepositorySQLite3Impl(ctx, localCacheDBFileName, k.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new kmemo rep: %w", err)
		return err
	}
	k.localCachedRep = newLocalCachedRep
	return nil
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return k.originalRep.GetRepName(ctx)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	k.m.Lock()
	defer k.m.Unlock()
	err := k.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	err = k.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	return nil
}

func (k *kmemoRepositorySQLite3ImplLocalCached) FindKmemo(ctx context.Context, query *find.FindQuery) ([]*Kmemo, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.FindKmemo(ctx, query)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetKmemo(ctx context.Context, id string, updateTime *time.Time) (*Kmemo, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.GetKmemo(ctx, id, updateTime)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetKmemoHistories(ctx context.Context, id string) ([]*Kmemo, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.GetKmemoHistories(ctx, id)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) AddKmemoInfo(ctx context.Context, kmemo *Kmemo) error {
	k.m.Lock()
	defer k.m.Unlock()
	err := k.originalRep.AddKmemoInfo(ctx, kmemo)
	if err != nil {
		return err
	}
	return k.UpdateCache(ctx)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]KmemoRepository, error) {
	return []KmemoRepository{k.originalRep}, nil
}

func (k *kmemoRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{k.originalRep}, nil
}
