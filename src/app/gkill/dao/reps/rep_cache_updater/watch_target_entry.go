package rep_cache_updater

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
)

var (
	startedService = false
)

type watchTargetEntry struct {
	rep                CacheUpdatable
	filename           string
	ignoreFilePrefixes []string

	watcher    *fsnotify.Watcher
	watchUsers map[string]struct{}

	skip *bool
}

func newWatchTargetEntry(rep CacheUpdatable, filename string, ignoreFilePrefixes []string, skip *bool) (*watchTargetEntry, error) {
	var err error
	// ファイル監視を始める
	err = watcher.Add(filename)
	if err != nil {
		err = fmt.Errorf("error at add watch file. filename = %s: %w", filename, err)
		return nil, err
	}
	if !startedService {
		startedService = true
		done := threads.AllocateThread()
		go func() {
			defer done()
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						err := fmt.Errorf("file watch event is not ok")
						gkill_log.Debug.Print(err)
						return
					}

					// 無視対象だったら何もしない
					ignore := false
					for _, ignoreFilePrefix := range ignoreFilePrefixes {
						if strings.HasPrefix(filepath.ToSlash(event.Name), ignoreFilePrefix) {
							ignore = true
							break
						}
					}
					if ignore {
						continue
					}

					if *skip {
						continue
					}

					// 無視対象でなければキャッシュを更新する
					err := rep.UpdateCache(context.TODO())
					if err != nil {
						err = fmt.Errorf("error at update cache. filename = %s: %w", filename, err)
						gkill_log.Debug.Print(err)
						return
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						err = fmt.Errorf("file watch event is not ok %w", err)
						gkill_log.Debug.Print(err)
						return
					}
					err = fmt.Errorf("file watch event is not ok %w", err)
					gkill_log.Debug.Print(err)
					return
				}
			}
		}()
	}
	return &watchTargetEntry{
		filename:           filename,
		rep:                rep,
		ignoreFilePrefixes: ignoreFilePrefixes,

		watcher: watcher,

		watchUsers: map[string]struct{}{},

		skip: skip,
	}, nil
}

func (w *watchTargetEntry) GetTargetFileName() (string, error) {
	return w.filename, nil
}

func (w *watchTargetEntry) GetWatchUsers() ([]string, error) {
	watchUsers := []string{}
	for watchUser := range w.watchUsers {
		watchUsers = append(watchUsers, watchUser)
	}
	return watchUsers, nil
}

func (w *watchTargetEntry) IsRegisteredUserID(userID string) (bool, error) {
	registeredUsers, err := w.GetWatchUsers()
	if err != nil {
		err = fmt.Errorf("error at get watching users: %w", err)
		return false, err
	}

	isRegisteredUserID := false
	for _, registeredUser := range registeredUsers {
		if registeredUser == userID {
			isRegisteredUserID = true
			break
		}
	}
	return isRegisteredUserID, nil
}

func (w *watchTargetEntry) AddWatchUser(userID string) error {
	w.watchUsers[userID] = struct{}{}
	return nil
}

func (w *watchTargetEntry) RemoveWatchUser(userID string) (isClosed bool, err error) {
	// あったら消す。
	_, exist := w.watchUsers[userID]
	if exist {
		delete(w.watchUsers, userID)
	}

	// まだ誰かが見ていたら何もせず返す
	if len(w.watchUsers) != 0 {
		return false, nil

	}
	return true, nil
}
