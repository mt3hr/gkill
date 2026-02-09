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

type lantanaRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          LantanaRepository
	localCachedRep       LantanaRepository
	m                    sync.Mutex

	fullConnect bool
}

func NewLantanaRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (LantanaRepository, error) {
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

	originalRep, err := NewLantanaRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new lantana rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewLantanaRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new lantana rep: %w", err)
		return nil, err
	}

	cachedRep := &lantanaRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.Mutex{},
	}
	return cachedRep, nil
}

func (l *lantanaRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	l.m.Lock()
	l.m.Unlock()
	return l.localCachedRep.FindKyous(ctx, query)
}

func (l *lantanaRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	l.m.Lock()
	l.m.Unlock()
	return l.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (l *lantanaRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	l.m.Lock()
	l.m.Unlock()
	return l.localCachedRep.GetKyouHistories(ctx, id)
}

func (l *lantanaRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	l.m.Lock()
	l.m.Unlock()
	return l.originalRep.GetPath(ctx, id)
}

func (l *lantanaRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	l.m.Lock()
	defer l.m.Unlock()

	err := l.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache %s", err)
		return err
	}

	err = os.Remove(l.localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at remove %s: %w", l.localCacheDBFileName, err)
		return err
	}

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.Replace(l.originalDBFileName, ":", "", -1))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(l.originalDBFileName)
	updateCache := originalStatErr != nil || cacheStatErr != nil || originalStat.ModTime().Equal(cacheStat.ModTime())
	if updateCache {
		originalDBFile, err := os.Open(l.originalDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", l.originalDBFileName)
		}
		defer originalDBFile.Close()
		cacheDBFile, err := os.Create(localCacheDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName)
		}
		defer cacheDBFile.Close()
		_, err = io.Copy(cacheDBFile, originalDBFile)
		if err != nil {
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", l.originalDBFileName, localCacheDBFileName, err)
			return err
		}
	}

	newLocalCachedRep, err := NewLantanaRepositorySQLite3Impl(ctx, localCacheDBFileName, l.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new lantana rep: %w", err)
		return err
	}
	l.localCachedRep = newLocalCachedRep
	return nil
}

func (l *lantanaRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return l.originalRep.GetRepName(ctx)
}

func (l *lantanaRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	err := l.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	err = l.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	return nil
}

func (l *lantanaRepositorySQLite3ImplLocalCached) FindLantana(ctx context.Context, query *find.FindQuery) ([]*Lantana, error) {
	l.m.Lock()
	l.m.Unlock()
	return l.localCachedRep.FindLantana(ctx, query)
}

func (l *lantanaRepositorySQLite3ImplLocalCached) GetLantana(ctx context.Context, id string, updateTime *time.Time) (*Lantana, error) {
	l.m.Lock()
	l.m.Unlock()
	return l.localCachedRep.GetLantana(ctx, id, updateTime)
}

func (l *lantanaRepositorySQLite3ImplLocalCached) GetLantanaHistories(ctx context.Context, id string) ([]*Lantana, error) {
	l.m.Lock()
	l.m.Unlock()
	return l.localCachedRep.GetLantanaHistories(ctx, id)
}

func (l *lantanaRepositorySQLite3ImplLocalCached) AddLantanaInfo(ctx context.Context, lantana *Lantana) error {
	l.m.Lock()
	defer l.m.Unlock()
	err := l.originalRep.AddLantanaInfo(ctx, lantana)
	if err != nil {
		return err
	}
	return l.UpdateCache(ctx)
}

func (l *lantanaRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]LantanaRepository, error) {
	return []LantanaRepository{l.originalRep}, nil
}

func (l *lantanaRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{l.originalRep}, nil
}
