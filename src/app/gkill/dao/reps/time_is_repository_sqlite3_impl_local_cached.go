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

type timeIsRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          TimeIsRepository
	localCachedRep       TimeIsRepository
	m                    sync.Mutex

	fullConnect bool
}

func NewTimeIsRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (TimeIsRepository, error) {
	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.ReplaceAll(filename, ":", ""))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err := os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return nil, err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(filename)
	updateCache := originalStatErr != nil || cacheStatErr != nil || originalStat.ModTime().Equal(cacheStat.ModTime())
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
	}

	originalRep, err := NewTimeIsRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new timeis rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewTimeIsRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new timeis rep: %w", err)
		return nil, err
	}

	cachedRep := &timeIsRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.Mutex{},
	}
	return cachedRep, nil
}
func (i *timeIsRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.FindKyous(ctx, query)
}

func (i *timeIsRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (i *timeIsRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.GetKyouHistories(ctx, id)
}

func (i *timeIsRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.originalRep.GetPath(ctx, id)
}

func (t *timeIsRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	t.m.Lock()
	defer t.m.Unlock()

	err := t.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache %s", err)
		return err
	}

	err = os.Remove(t.localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at remove %s: %w", t.localCacheDBFileName, err)
		return err
	}

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.Replace(t.originalDBFileName, ":", "", -1))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(t.originalDBFileName)
	updateCache := originalStatErr != nil || cacheStatErr != nil || originalStat.ModTime().Equal(cacheStat.ModTime())
	if updateCache {
		originalDBFile, err := os.Open(t.originalDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", t.originalDBFileName)
		}
		defer originalDBFile.Close()
		cacheDBFile, err := os.Create(localCacheDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName)
		}
		defer cacheDBFile.Close()
		_, err = io.Copy(cacheDBFile, originalDBFile)
		if err != nil {
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", t.originalDBFileName, localCacheDBFileName, err)
			return err
		}
	}

	newLocalCachedRep, err := NewTimeIsRepositorySQLite3Impl(ctx, localCacheDBFileName, t.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new timeis rep: %w", err)
		return err
	}
	t.localCachedRep = newLocalCachedRep
	return nil
}

func (i *timeIsRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return i.originalRep.GetRepName(ctx)
}

func (i *timeIsRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	err := i.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	err = i.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	return nil
}

func (i *timeIsRepositorySQLite3ImplLocalCached) FindTimeIs(ctx context.Context, query *find.FindQuery) ([]*TimeIs, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.FindTimeIs(ctx, query)
}

func (i *timeIsRepositorySQLite3ImplLocalCached) GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.GetTimeIs(ctx, id, updateTime)
}

func (i *timeIsRepositorySQLite3ImplLocalCached) GetTimeIsHistories(ctx context.Context, id string) ([]*TimeIs, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.GetTimeIsHistories(ctx, id)
}

func (i *timeIsRepositorySQLite3ImplLocalCached) AddTimeIsInfo(ctx context.Context, timeis *TimeIs) error {
	i.m.Lock()
	defer i.m.Unlock()
	err := i.originalRep.AddTimeIsInfo(ctx, timeis)
	if err != nil {
		return err
	}
	return i.UpdateCache(ctx)
}

func (i *timeIsRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]TimeIsRepository, error) {
	return []TimeIsRepository{i.originalRep}, nil
}

func (i *timeIsRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{i.originalRep}, nil
}
