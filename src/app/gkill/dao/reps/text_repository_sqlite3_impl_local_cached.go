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

type textRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          TextRepository
	localCachedRep       TextRepository
	m                    sync.Mutex

	fullConnect bool
}

func NewTextRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (TextRepository, error) {
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

	originalRep, err := NewTextRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new text rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewTextRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new text rep: %w", err)
		return nil, err
	}

	cachedRep := &textRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.Mutex{},
	}
	return cachedRep, nil
}
func (t *textRepositorySQLite3ImplLocalCached) FindTexts(ctx context.Context, query *find.FindQuery) ([]*Text, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.FindTexts(ctx, query)
}

func (t *textRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	err := t.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	err = t.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close %s", err)
		return err
	}
	return nil
}

func (t *textRepositorySQLite3ImplLocalCached) GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.GetText(ctx, id, updateTime)
}

func (t *textRepositorySQLite3ImplLocalCached) GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.GetTextsByTargetID(ctx, target_id)
}

func (t *textRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
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

	newLocalCachedRep, err := NewTextRepositorySQLite3Impl(ctx, localCacheDBFileName, t.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new text rep: %w", err)
		return err
	}
	t.localCachedRep = newLocalCachedRep
	return nil
}

func (t *textRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.originalRep.GetPath(ctx, id)
}

func (t *textRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return t.originalRep.GetRepName(ctx)
}

func (t *textRepositorySQLite3ImplLocalCached) GetTextHistories(ctx context.Context, id string) ([]*Text, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.GetTextHistories(ctx, id)
}

func (t *textRepositorySQLite3ImplLocalCached) AddTextInfo(ctx context.Context, text *Text) error {
	t.m.Lock()
	defer t.m.Unlock()
	err := t.originalRep.AddTextInfo(ctx, text)
	if err != nil {
		return err
	}
	return t.UpdateCache(ctx)
}

func (t *textRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]TextRepository, error) {
	return []TextRepository{t.originalRep}, nil
}
