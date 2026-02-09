package reps

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

func NewIDFDirRepLocalCached(ctx context.Context, dir, dbFilename string, fullConnect bool, r *mux.Router, autoIDF bool, idfIgnore *[]string, repositoriesRef *GkillRepositories) (IDFKyouRepository, error) {
	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.ReplaceAll(dbFilename, ":", ""))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err := os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return nil, err
	}

	originalDBFile, err := os.Open(dbFilename)
	if err != nil {
		err = fmt.Errorf("error at open file %s: %w", dbFilename)
	}
	defer originalDBFile.Close()
	cacheDBFile, err := os.Create(localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName)
	}
	defer cacheDBFile.Close()
	_, err = io.Copy(cacheDBFile, originalDBFile)
	if err != nil {
		err = fmt.Errorf("error at copy local cache db %s to %s: %w", dbFilename, localCacheDBFileName, err)
		return nil, err
	}

	originalRep, err := NewIDFDirRep(ctx, dir, dbFilename, false, r, autoIDF, idfIgnore, repositoriesRef)
	if err != nil {
		err = fmt.Errorf("error at new idf dir rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewIDFDirRep(ctx, dir, localCacheDBFileName, fullConnect, r, false, idfIgnore, repositoriesRef)
	if err != nil {
		err = fmt.Errorf("error at new idf dir rep: %w", err)
		return nil, err
	}

	cachedRep := &idfKyouRepositorySQLite3ImplLocalCached{
		originalDBFileName:   dbFilename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		repositoriesRef: repositoriesRef,
		r:               r,
		contentDir:      dir,
		fullConnect:     fullConnect,
		autoIDF:         autoIDF,
		idfIgnore:       idfIgnore,

		m: sync.Mutex{},
	}
	return cachedRep, nil
}

type idfKyouRepositorySQLite3ImplLocalCached struct {
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          IDFKyouRepository
	localCachedRep       IDFKyouRepository
	m                    sync.Mutex

	repositoriesRef *GkillRepositories
	r               *mux.Router
	contentDir      string
	fullConnect     bool
	autoIDF         bool
	idfIgnore       *[]string
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.FindKyous(ctx, query)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.GetKyouHistories(ctx, id)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.originalRep.GetPath(ctx, id)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	i.m.Lock()
	defer i.m.Unlock()

	err := i.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache %s", err)
		return err
	}

	err = os.Remove(i.localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at remove %s: %w", i.localCacheDBFileName, err)
		return err
	}

	localCacheDBFileName := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_cache_rep", strings.Replace(i.originalDBFileName, ":", "", -1))
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	originalDBFile, err := os.Open(i.originalDBFileName)
	if err != nil {
		err = fmt.Errorf("error at open file %s: %w", i.originalDBFileName)
	}
	defer originalDBFile.Close()
	cacheDBFile, err := os.Create(localCacheDBFileName)
	if err != nil {
		err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName)
	}
	defer cacheDBFile.Close()
	_, err = io.Copy(cacheDBFile, originalDBFile)
	if err != nil {
		err = fmt.Errorf("error at copy local cache db %s to %s: %w", i.originalDBFileName, localCacheDBFileName, err)
		return err
	}

	newLocalCachedRep, err := NewIDFDirRep(ctx, i.contentDir, localCacheDBFileName, i.fullConnect, i.r, i.autoIDF, i.idfIgnore, i.repositoriesRef)
	if err != nil {
		err = fmt.Errorf("error at new idf dir rep: %w", err)
		return err
	}
	i.localCachedRep = newLocalCachedRep
	return nil
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return i.originalRep.GetRepName(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
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

func (i *idfKyouRepositorySQLite3ImplLocalCached) FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]*IDFKyou, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.FindIDFKyou(ctx, query)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.GetIDFKyou(ctx, id, updateTime)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetIDFKyouHistories(ctx context.Context, id string) ([]*IDFKyou, error) {
	i.m.Lock()
	i.m.Unlock()
	return i.localCachedRep.GetIDFKyouHistories(ctx, id)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) IDF(ctx context.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	err := i.originalRep.IDF(ctx)
	if err != nil {
		return err
	}
	return i.UpdateCache(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) AddIDFKyouInfo(ctx context.Context, idfKyou *IDFKyou) error {
	i.m.Lock()
	defer i.m.Unlock()
	err := i.originalRep.AddIDFKyouInfo(ctx, idfKyou)
	if err != nil {
		return err
	}
	return i.UpdateCache(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) HandleFileServe(w http.ResponseWriter, r *http.Request) {
	i.originalRep.HandleFileServe(w, r)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GenerateThumbCache(ctx context.Context) error {
	return i.originalRep.GenerateThumbCache(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) ClearThumbCache() error {
	return i.originalRep.ClearThumbCache()
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GenerateVideoCache(ctx context.Context) error {
	return i.originalRep.GenerateVideoCache(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) ClearVideoCache() error {
	return i.originalRep.ClearVideoCache()
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]IDFKyouRepository, error) {
	return []IDFKyouRepository{i.originalRep}, nil
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{i.originalRep}, nil
}
