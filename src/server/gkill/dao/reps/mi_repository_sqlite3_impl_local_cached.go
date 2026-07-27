package reps

import (
	"context"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

type miRepositorySQLite3ImplLocalCached struct {
	userID               string
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          MiRepository
	localCachedRep       MiRepository
	m                    sync.RWMutex

	fullConnect bool

	lastUpdateCacheChanged bool
}

func NewMiRepositorySQLite3ImplLocalCached(ctx context.Context, userID string, filename string, fullConnect bool) (MiRepository, error) {
	localCacheDBFileName := localRepCacheDBFileName(userID, filename)
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
		userID:               userID,
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}

func (m *miRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	if query.UpdateCache {
		if err := m.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.FindKyous(ctx, query)
}

func (m *miRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (m *miRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
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
		err = fmt.Errorf("error at update cache: %w", err)
		return err
	}

	// ローカルキャッシュファイルを消せなくても致命的なエラーにはしない。
	// 別のRepositoryや別プロセスが同じファイルを開いていると
	// Windowsでは削除が共有違反で失敗する。
	// ここで error を返すと GetRepositories 全体が失敗し、
	// リポジトリがキャッシュに登録されないままリクエスト毎に再ロードされ続けるため、
	// この回の再取得だけ諦めて、閉じたハンドルを開き直して継続する。
	if err = os.Remove(m.localCacheDBFileName); err != nil && !os.IsNotExist(err) {
		slog.Log(ctx, gkill_log.Warn, "skip local rep cache refresh", "file", m.localCacheDBFileName, "error", err)
		m.lastUpdateCacheChanged = false
		reopenedRep, reopenErr := NewMiRepositorySQLite3Impl(ctx, m.localCacheDBFileName, m.fullConnect)
		if reopenErr != nil {
			return fmt.Errorf("error at reopen local cached rep %s: %w", m.localCacheDBFileName, reopenErr)
		}
		m.localCachedRep = reopenedRep
		return nil
	}

	localCacheDBFileName := localRepCacheDBFileName(m.userID, m.originalDBFileName)
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(m.originalDBFileName)
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
	m.lastUpdateCacheChanged = updateCache
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

func (m *miRepositorySQLite3ImplLocalCached) LastUpdateCacheChanged() bool {
	return m.lastUpdateCacheChanged
}

func (m *miRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.originalRep.GetRepName(ctx)
}

func (m *miRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	err := m.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	err = m.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	return nil
}

func (m *miRepositorySQLite3ImplLocalCached) FindMi(ctx context.Context, query *find.FindQuery) ([]Mi, error) {
	if query.UpdateCache {
		if err := m.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.FindMi(ctx, query)
}

func (m *miRepositorySQLite3ImplLocalCached) GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetMi(ctx, id, updateTime)
}

func (m *miRepositorySQLite3ImplLocalCached) GetMiHistories(ctx context.Context, id string) ([]Mi, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetMiHistories(ctx, id)
}

func (m *miRepositorySQLite3ImplLocalCached) AddMiInfo(ctx context.Context, mi Mi) error {
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

func (m *miRepositorySQLite3ImplLocalCached) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	addrs, err := m.localCachedRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	repName, err := m.GetRepName(ctx)
	if err != nil {
		return nil, err
	}
	for idx := range addrs {
		addrs[idx].LatestDataRepositoryName = repName
	}
	return addrs, nil
}
