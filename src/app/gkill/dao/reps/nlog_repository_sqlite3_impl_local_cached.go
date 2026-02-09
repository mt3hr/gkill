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

type nlogRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          NlogRepository
	localCachedRep       NlogRepository
	m                    sync.Mutex

	fullConnect bool
}

func NewNlogRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (NlogRepository, error) {
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

	originalRep, err := NewNlogRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new nlog rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewNlogRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new nlog rep: %w", err)
		return nil, err
	}

	cachedRep := &nlogRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.Mutex{},
	}
	return cachedRep, nil
}
func (n *nlogRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	n.m.Lock()
	n.m.Unlock()
	return n.localCachedRep.FindKyous(ctx, query)
}

func (n *nlogRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	n.m.Lock()
	n.m.Unlock()
	return n.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (n *nlogRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	n.m.Lock()
	n.m.Unlock()
	return n.localCachedRep.GetKyouHistories(ctx, id)
}

func (n *nlogRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	n.m.Lock()
	n.m.Unlock()
	return n.originalRep.GetPath(ctx, id)
}

func (n *nlogRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
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

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.Replace(n.originalDBFileName, ":", "", -1))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(n.originalDBFileName)
	updateCache := originalStatErr != nil || cacheStatErr != nil || originalStat.ModTime().Equal(cacheStat.ModTime())
	if updateCache {
		originalDBFile, err := os.Open(n.originalDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", n.originalDBFileName)
		}
		defer originalDBFile.Close()
		cacheDBFile, err := os.Create(localCacheDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName)
		}
		defer cacheDBFile.Close()
		_, err = io.Copy(cacheDBFile, originalDBFile)
		if err != nil {
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", n.originalDBFileName, localCacheDBFileName, err)
			return err
		}
	}

	newLocalCachedRep, err := NewNlogRepositorySQLite3Impl(ctx, localCacheDBFileName, n.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new nlog rep: %w", err)
		return err
	}
	n.localCachedRep = newLocalCachedRep
	return nil
}

func (n *nlogRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return n.originalRep.GetRepName(ctx)
}

func (n *nlogRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
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

func (n *nlogRepositorySQLite3ImplLocalCached) FindNlog(ctx context.Context, query *find.FindQuery) ([]*Nlog, error) {
	n.m.Lock()
	n.m.Unlock()
	return n.localCachedRep.FindNlog(ctx, query)
}

func (n *nlogRepositorySQLite3ImplLocalCached) GetNlog(ctx context.Context, id string, updateTime *time.Time) (*Nlog, error) {
	n.m.Lock()
	n.m.Unlock()
	return n.localCachedRep.GetNlog(ctx, id, updateTime)
}

func (n *nlogRepositorySQLite3ImplLocalCached) GetNlogHistories(ctx context.Context, id string) ([]*Nlog, error) {
	n.m.Lock()
	n.m.Unlock()
	return n.localCachedRep.GetNlogHistories(ctx, id)
}

func (n *nlogRepositorySQLite3ImplLocalCached) AddNlogInfo(ctx context.Context, nlog *Nlog) error {
	n.m.Lock()
	defer n.m.Unlock()
	err := n.originalRep.AddNlogInfo(ctx, nlog)
	if err != nil {
		return err
	}
	return n.UpdateCache(ctx)
}

func (n *nlogRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]NlogRepository, error) {
	return []NlogRepository{n.originalRep}, nil
}

func (n *nlogRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{n.originalRep}, nil
}
