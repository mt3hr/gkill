package rep_cache_updater

import (
	"fmt"
	"sync"
)

func NewFileRepCacheUpdater(skip *bool) (FileRepCacheUpdater, error) {
	return &fileRepCacheUpdaterImpl{
		watchTargets: map[string]*watchTargetEntry{},
		skip:         skip,
	}, nil
}

type fileRepCacheUpdaterImpl struct {
	m            sync.Mutex
	watchTargets map[string]*watchTargetEntry // map[対象ファイル名] = 監視対象の情報
	skip         *bool
}

func (f *fileRepCacheUpdaterImpl) RegisterWatchFileRep(rep CacheUpdatable, filename string, ignoreFilePrefixes []string, userID string) error {
	var err error

	f.m.Lock()
	defer f.m.Unlock()

	target, exist := f.watchTargets[filename]
	if !exist {
		// なかったら作って監視を開始する
		target, err = newWatchTargetEntry(rep, filename, ignoreFilePrefixes, f.skip)
		if err != nil {
			err = fmt.Errorf("error at new watch target entry: %w", err)
			return err
		}
		f.watchTargets[filename] = target
	}

	// 登録されているユーザにあれば何もしない。
	isRegistered, err := target.IsRegisteredUserID(userID)
	if err != nil {
		err = fmt.Errorf("error at get is registered user id: %w", err)
		return err
	}

	// まだなければ登録する
	if isRegistered {
		return nil
	}

	err = target.AddWatchUser(userID)
	if err != nil {
		err = fmt.Errorf("error at add watch user. user id = %s filename = %s: %w", userID, filename, err)
		return err
	}

	return nil
}

func (f *fileRepCacheUpdaterImpl) RemoveWatchFileRep(filename string, userID string) error {
	var err error

	f.m.Lock()
	defer f.m.Unlock()

	target, exist := f.watchTargets[filename]
	if !exist {
		return nil
	}

	// 登録されていなければ何もしないで返す
	isRegistered, err := target.IsRegisteredUserID(userID)
	if err != nil {
		err = fmt.Errorf("error at get is registered user id: %w", err)
		return err
	}

	if !isRegistered {
		return nil
	}

	isClosed, err := target.RemoveWatchUser(userID)
	if err != nil {
		err = fmt.Errorf("error at remove watch user. user id = %s: %w", userID, err)
		return err
	}

	// ファイル監視が終わったらMapからも消す
	if isClosed {
		delete(f.watchTargets, filename)
	}
	return nil
}

func (f *fileRepCacheUpdaterImpl) Close() error {
	var err error
	for _, watchTarget := range f.watchTargets {
		if watchTarget == nil || watchTarget.watcher == nil {
			continue
		}
		err = watchTarget.watcher.Close()
		if err != nil {
			targetFileName, _ := watchTarget.GetTargetFileName()
			err = fmt.Errorf("error at close file rep cache updater impl. targetfilename = %s: %w", targetFileName, err)
		}
	}

	if err != nil {
		return err
	}

	return nil

}
