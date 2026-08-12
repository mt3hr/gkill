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

type tagRepositorySQLite3ImplLocalCached struct {
	userID               string
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          TagRepository
	localCachedRep       TagRepository
	m                    sync.RWMutex

	fullConnect bool
	// cacheChange は元DBファイルが前回のキャッシュ再構築から変わったかを見ます。
	// 上位のキャッシュrepはこれが false のときフルリビルドを飛ばします。
	cacheChange *dbFileChangeDetector

	// skipRebuild はローカルキャッシュを更新できなかった回だけ立ちます。
	// 古いコピーのまま上位に再構築させると、そのあと基準が進んで
	// 新しい内容が二度と取り込まれなくなるため、その回だけ再構築を見送らせます。
	skipRebuild bool
}

func NewTagRepositorySQLite3ImplLocalCached(ctx context.Context, userID string, filename string, fullConnect bool) (TagRepository, error) {
	localCacheDBFileName, err := localRepCacheDBFileName(userID, filename)
	if err != nil {
		return nil, err
	}
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
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
		userID:               userID,
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		cacheChange: &dbFileChangeDetector{},

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}
func (t *tagRepositorySQLite3ImplLocalCached) FindTags(ctx context.Context, query *find.FindQuery) ([]Tag, error) {
	if query.UpdateCache {
		if err := t.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	t.m.RLock()
	defer t.m.RUnlock()
	return t.localCachedRep.FindTags(ctx, query)
}

func (t *tagRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	t.m.Lock()
	defer t.m.Unlock()
	err := t.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	err = t.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	return nil
}

func (t *tagRepositorySQLite3ImplLocalCached) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.localCachedRep.GetTag(ctx, id, updateTime)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetTagsByTagName(ctx context.Context, tagname string) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.localCachedRep.GetTagsByTagName(ctx, tagname)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetTagsByTargetID(ctx context.Context, target_id string) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.localCachedRep.GetTagsByTargetID(ctx, target_id)
}

func (t *tagRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	t.m.Lock()
	defer t.m.Unlock()

	// 上位のキャッシュrepへ「元DBが変わったか」を伝えるための観測。
	// 基準が進むのは再構築が成功したときのCommitCacheRebuildだけなので、
	// 途中で失敗した回のぶんは次回も再構築される。
	t.cacheChange.refresh(t.originalDBFileName)
	t.skipRebuild = false

	// 元DBがローカルキャッシュと同じなら、閉じる・消す・コピーし直す・開き直すを全部飛ばす。
	// この判定を os.Remove のあとに置くと、消した直後なので必ず「要コピー」になり、
	// 変更のないrepまで LastUpdateCacheChanged() が true を返して
	// 上位のキャッシュrepが毎回フルリビルドする。
	if !localRepCacheNeedsCopy(t.originalDBFileName, t.localCacheDBFileName) {
		return nil
	}

	err := t.localCachedRep.Close(ctx)
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
	if err = os.Remove(t.localCacheDBFileName); err != nil && !os.IsNotExist(err) {
		slog.Log(ctx, gkill_log.Warn, "skip local rep cache refresh", "file", fmt.Sprintf("%q", t.localCacheDBFileName), "error", fmt.Sprintf("%q", err))
		t.skipRebuild = true
		reopenedRep, reopenErr := NewTagRepositorySQLite3Impl(ctx, t.localCacheDBFileName, t.fullConnect)
		if reopenErr != nil {
			return fmt.Errorf("error at reopen local cached rep %s: %w", t.localCacheDBFileName, reopenErr)
		}
		t.localCachedRep = reopenedRep
		return nil
	}

	localCacheDBFileName, err := localRepCacheDBFileName(t.userID, t.originalDBFileName)
	if err != nil {
		return err
	}
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	// ここへ来るのは冒頭の判定でコピーが要ると分かったときだけ。
	if err := copyLocalRepCacheDB(t.originalDBFileName, localCacheDBFileName); err != nil {
		return err
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
	t.m.RLock()
	defer t.m.RUnlock()
	return t.originalRep.GetPath(ctx, id)
}

func (t *tagRepositorySQLite3ImplLocalCached) LastUpdateCacheChanged() bool {
	// ローカルキャッシュを更新できなかった回は、古いコピーから作り直させない。
	if t.skipRebuild {
		return false
	}
	return t.cacheChange.lastChanged()
}

// CommitCacheRebuild は上位のキャッシュrepが再構築に成功したときに呼ばれます。
func (t *tagRepositorySQLite3ImplLocalCached) CommitCacheRebuild() {
	t.cacheChange.commit()
}

func (t *tagRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return t.originalRep.GetRepName(ctx)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetTagHistories(ctx context.Context, id string) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.localCachedRep.GetTagHistories(ctx, id)
}

func (t *tagRepositorySQLite3ImplLocalCached) AddTagInfo(ctx context.Context, tag Tag) error {
	err := func() error {
		t.m.Lock()
		defer t.m.Unlock()
		err := t.originalRep.AddTagInfo(ctx, tag)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}
	return t.UpdateCache(ctx)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetAllTagNames(ctx context.Context) ([]string, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.localCachedRep.GetAllTagNames(ctx)
}

func (t *tagRepositorySQLite3ImplLocalCached) GetAllTags(ctx context.Context) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.localCachedRep.GetAllTags(ctx)
}

func (t *tagRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]TagRepository, error) {
	return []TagRepository{t.originalRep}, nil
}

func (t *tagRepositorySQLite3ImplLocalCached) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	addrs, err := t.localCachedRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	repName, err := t.GetRepName(ctx)
	if err != nil {
		return nil, err
	}
	for idx := range addrs {
		addrs[idx].LatestDataRepositoryName = repName
	}
	return addrs, nil
}
