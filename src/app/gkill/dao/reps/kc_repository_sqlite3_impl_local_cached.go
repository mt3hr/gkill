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

type kcRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          KCRepository
	localCachedRep       KCRepository
	m                    sync.Mutex

	fullConnect bool
}

func NewKCRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (KCRepository, error) {
	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.ReplaceAll(filename, ":", ""))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err := os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return nil, err
	}

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

	originalRep, err := NewKCRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new kc rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewKCRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new kc rep: %w", err)
		return nil, err
	}

	cachedRep := &kcRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.Mutex{},
	}
	return cachedRep, nil
}

func (k *kcRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	k.m.Lock()
	k.m.Unlock()
	return k.localCachedRep.FindKyous(ctx, query)
}

func (k *kcRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	k.m.Lock()
	k.m.Unlock()
	return k.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (k *kcRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	k.m.Lock()
	k.m.Unlock()
	return k.localCachedRep.GetKyouHistories(ctx, id)
}

func (k *kcRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	k.m.Lock()
	k.m.Unlock()
	return k.originalRep.GetPath(ctx, id)
}

func (k *kcRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
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

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.Replace(k.originalDBFileName, ":", "", -1))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	originalDBFile, err := os.Open(k.originalDBFileName)
	if err != nil {
		err = fmt.Errorf("error at open file %s: %w", k.originalDBFileName)
	}
	defer originalDBFile.Close()
	cacheDBFile, err := os.Create(localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName)
	}
	defer cacheDBFile.Close()
	_, err = io.Copy(cacheDBFile, originalDBFile)
	if err != nil {
		err = fmt.Errorf("error at copy local cache db %s to %s: %w", k.originalDBFileName, localCacheDBFileName, err)
		return err
	}

	newLocalCachedRep, err := NewKCRepositorySQLite3Impl(ctx, localCacheDBFileName, k.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new kc rep: %w", err)
		return err
	}
	k.localCachedRep = newLocalCachedRep
	return nil
}

func (k *kcRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return k.originalRep.GetRepName(ctx)
}

func (k *kcRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
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

func (k *kcRepositorySQLite3ImplLocalCached) FindKC(ctx context.Context, query *find.FindQuery) ([]*KC, error) {
	k.m.Lock()
	k.m.Unlock()
	return k.localCachedRep.FindKC(ctx, query)
}

func (k *kcRepositorySQLite3ImplLocalCached) GetKC(ctx context.Context, id string, updateTime *time.Time) (*KC, error) {
	k.m.Lock()
	k.m.Unlock()
	return k.localCachedRep.GetKC(ctx, id, updateTime)
}

func (k *kcRepositorySQLite3ImplLocalCached) GetKCHistories(ctx context.Context, id string) ([]*KC, error) {
	k.m.Lock()
	k.m.Unlock()
	return k.localCachedRep.GetKCHistories(ctx, id)
}

func (k *kcRepositorySQLite3ImplLocalCached) AddKCInfo(ctx context.Context, kc *KC) error {
	k.m.Lock()
	defer k.m.Unlock()
	err := k.originalRep.AddKCInfo(ctx, kc)
	if err != nil {
		return err
	}
	return k.UpdateCache(ctx)
}

func (k *kcRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]KCRepository, error) {
	return []KCRepository{k.originalRep}, nil
}

func (k *kcRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{k.originalRep}, nil
}
