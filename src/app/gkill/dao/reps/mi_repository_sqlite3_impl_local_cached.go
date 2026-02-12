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

type miRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          MiRepository
	localCachedRep       MiRepository
	m                    sync.RWMutex

	fullConnect bool
}

func NewMiRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (MiRepository, error) {
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

	originalRep, err := NewMiRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new mi rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewMiRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new mi rep: %w", err)
		return nil, err
	}

	cachedRep := &miRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}

func (m *miRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.FindKyous(ctx, query)
}

func (m *miRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (m *miRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetKyouHistories(ctx, id)
}

func (m *miRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.originalRep.GetPath(ctx, id)
}

func (m *miRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()

	err := m.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache %s", err)
		return err
	}

	err = os.Remove(m.localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at remove %s: %w", m.localCacheDBFileName, err)
		return err
	}

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.ReplaceAll(m.originalDBFileName, ":", ""))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(m.originalDBFileName)
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
	if updateCache {
		originalDBFile, err := os.Open(m.originalDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", m.originalDBFileName, err)
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
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", m.originalDBFileName, localCacheDBFileName, err)
			return err
		}
		os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
	}

	newLocalCachedRep, err := NewMiRepositorySQLite3Impl(ctx, localCacheDBFileName, m.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new mi rep: %w", err)
		return err
	}
	m.localCachedRep = newLocalCachedRep
	return nil
}

func (m *miRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.originalRep.GetRepName(ctx)
}

func (m *miRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	err := m.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	err = m.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	return nil
}

func (m *miRepositorySQLite3ImplLocalCached) FindMi(ctx context.Context, query *find.FindQuery) ([]*Mi, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.FindMi(ctx, query)
}

func (m *miRepositorySQLite3ImplLocalCached) GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetMi(ctx, id, updateTime)
}

func (m *miRepositorySQLite3ImplLocalCached) GetMiHistories(ctx context.Context, id string) ([]*Mi, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetMiHistories(ctx, id)
}

func (m *miRepositorySQLite3ImplLocalCached) AddMiInfo(ctx context.Context, mi *Mi) error {
	err := func() error {
		m.m.Lock()
		defer m.m.Unlock()
		err := m.originalRep.AddMiInfo(ctx, mi)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}
	return m.UpdateCache(ctx)
}

func (m *miRepositorySQLite3ImplLocalCached) GetBoardNames(ctx context.Context) ([]string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.originalRep.GetBoardNames(ctx)
}

func (m *miRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]MiRepository, error) {
	return []MiRepository{m.originalRep}, nil
}

func (m *miRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{m.originalRep}, nil
}
