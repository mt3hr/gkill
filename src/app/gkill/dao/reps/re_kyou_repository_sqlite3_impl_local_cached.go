package reps

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type reKyouRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          ReKyouRepository
	localCachedRep       ReKyouRepository
	m                    sync.Mutex

	fullConnect bool
	reps        *GkillRepositories
}

func NewReKyouRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool, reps *GkillRepositories) (ReKyouRepository, error) {
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
		}
		defer originalDBFile.Close()
		cacheDBFile, err := os.Create(localCacheDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName, err)
		}
		defer cacheDBFile.Close()
		_, err = io.Copy(cacheDBFile, originalDBFile)
		if err != nil {
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", filename, localCacheDBFileName, err)
			return nil, err
		}
		os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
	}

	originalRep, err := NewReKyouRepositorySQLite3Impl(ctx, filename, false, reps)
	if err != nil {
		err = fmt.Errorf("error at new rekyou rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewReKyouRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect, reps)
	if err != nil {
		err = fmt.Errorf("error at new rekyou rep: %w", err)
		return nil, err
	}

	cachedRep := &reKyouRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,
		reps:        reps,

		m: sync.Mutex{},
	}
	return cachedRep, nil
}

func (r *reKyouRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	r.m.Lock()
	r.m.Unlock()
	return r.localCachedRep.FindKyous(ctx, query)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	r.m.Lock()
	r.m.Unlock()
	return r.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	r.m.Lock()
	r.m.Unlock()
	return r.localCachedRep.GetKyouHistories(ctx, id)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	r.m.Lock()
	r.m.Unlock()
	return r.originalRep.GetPath(ctx, id)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	r.m.Lock()
	defer r.m.Unlock()

	err := r.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache %s", err)
		return err
	}

	err = os.Remove(r.localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at remove %s: %w", r.localCacheDBFileName, err)
		return err
	}

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.Replace(r.originalDBFileName, ":", "", -1))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(r.originalDBFileName)
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
	if updateCache {
		originalDBFile, err := os.Open(r.originalDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", r.originalDBFileName, err)
		}
		defer originalDBFile.Close()
		cacheDBFile, err := os.Create(localCacheDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName, err)
		}
		defer cacheDBFile.Close()
		_, err = io.Copy(cacheDBFile, originalDBFile)
		if err != nil {
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", r.originalDBFileName, localCacheDBFileName, err)
			return err
		}
		os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
	}

	newLocalCachedRep, err := NewReKyouRepositorySQLite3Impl(ctx, localCacheDBFileName, r.fullConnect, r.reps)
	if err != nil {
		err = fmt.Errorf("error at new rekyou rep: %w", err)
		return err
	}
	r.localCachedRep = newLocalCachedRep
	return nil
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return r.originalRep.GetRepName(ctx)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	err := r.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	err = r.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	return nil
}

func (r *reKyouRepositorySQLite3ImplLocalCached) FindReKyou(ctx context.Context, query *find.FindQuery) ([]*ReKyou, error) {
	r.m.Lock()
	r.m.Unlock()
	return r.localCachedRep.FindReKyou(ctx, query)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error) {
	r.m.Lock()
	r.m.Unlock()
	return r.localCachedRep.GetReKyou(ctx, id, updateTime)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetReKyouHistories(ctx context.Context, id string) ([]*ReKyou, error) {
	r.m.Lock()
	r.m.Unlock()
	return r.localCachedRep.GetReKyouHistories(ctx, id)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) AddReKyouInfo(ctx context.Context, rekyou *ReKyou) error {
	r.m.Lock()
	defer r.m.Unlock()
	err := r.originalRep.AddReKyouInfo(ctx, rekyou)
	if err != nil {
		return err
	}
	return r.UpdateCache(ctx)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetReKyousAllLatest(ctx context.Context) ([]*ReKyou, error) {
	r.m.Lock()
	r.m.Unlock()
	return r.localCachedRep.GetReKyousAllLatest(ctx)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	r.m.Lock()
	r.m.Unlock()
	return r.localCachedRep.GetRepositoriesWithoutReKyouRep(ctx)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]ReKyouRepository, error) {
	return []ReKyouRepository{r.originalRep}, nil
}

func (r *reKyouRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{r.originalRep}, nil
}
