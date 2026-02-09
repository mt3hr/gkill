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

type tagRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          TagRepository
	localCachedRep       TagRepository
	m                    sync.Mutex

	fullConnect bool
}

func NewTagRepositorySQLite3ImplLocalCached(ctx context.Context, filename string, fullConnect bool) (TagRepository, error) {
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
	}

	originalRep, err := NewTagRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new tag rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewTagRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new tag rep: %w", err)
		return nil, err
	}

	cachedRep := &tagRepositorySQLite3ImplLocalCached{
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		m: sync.Mutex{},
	}
	return cachedRep, nil
}
func (t *tagRepositorySQLite3ImplLocalCached) FindTags(ctx context.Context, query *find.FindQuery) ([]*Tag, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.FindTags(ctx, query)
}

func (t *tagRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
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

func (t *tagRepositorySQLite3ImplLocalCached) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.GetTag(ctx, id, updateTime)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.GetTagsByTagName(ctx, tagname)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.GetTagsByTargetID(ctx, target_id)
}

func (t *tagRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
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
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
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

	newLocalCachedRep, err := NewTagRepositorySQLite3Impl(ctx, localCacheDBFileName, t.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new tag rep: %w", err)
		return err
	}
	t.localCachedRep = newLocalCachedRep
	return nil
}

func (t *tagRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.originalRep.GetPath(ctx, id)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return t.originalRep.GetRepName(ctx)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetTagHistories(ctx context.Context, id string) ([]*Tag, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.GetTagHistories(ctx, id)
}

func (t *tagRepositorySQLite3ImplLocalCached) AddTagInfo(ctx context.Context, tag *Tag) error {
	t.m.Lock()
	defer t.m.Unlock()
	err := t.originalRep.AddTagInfo(ctx, tag)
	if err != nil {
		return err
	}
	return t.UpdateCache(ctx)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetAllTagNames(ctx context.Context) ([]string, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.GetAllTagNames(ctx)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetAllTags(ctx context.Context) ([]*Tag, error) {
	t.m.Lock()
	t.m.Unlock()
	return t.localCachedRep.GetAllTags(ctx)
}

func (t *tagRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]TagRepository, error) {
	return []TagRepository{t.originalRep}, nil
}
