package rep_cache_updater

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type watchTargetEntry struct {
	rep                CacheUpdatable
	filename           string
	ignoreFilePrefixes []string

	watcher        *fsnotify.Watcher
	requestCloseCh chan interface{}
	watchUsers     map[string]struct{}
}

func newWatchTargetEntry(rep CacheUpdatable, filename string, ignoreFilePrefixes []string) (*watchTargetEntry, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		err = fmt.Errorf("error at new watcher: %w", err)
		return nil, err
	}

	// ファイル監視を始める
	err = watcher.Add(filename)
	if err != nil {
		err = fmt.Errorf("error at add watch file. filename = %s: %w", filename, err)
		return nil, err
	}
	requestCloseCh := make(chan interface{}, 1) // goroutine終了用Ch
	go func() {
		for {
			select {
			case <-requestCloseCh:
				// 誰も見なくなったときにファイルの監視を終了する
				err := watcher.Close()
				if err != nil {
					fmt.Errorf("error at close watcher: %w", err)
					gkill_log.Debug.Fatal(err)
					return
				}
				return
			case event, ok := <-watcher.Events:
				if !ok {
					err := fmt.Errorf("file watch event is not ok")
					gkill_log.Debug.Fatal(err)
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

				// 無視対象でなければキャッシュを更新する
				err := rep.UpdateCache(context.TODO())
				if err != nil {
					err = fmt.Errorf("error at update cache. filename = %s: %w", filename, err)
					gkill_log.Debug.Print(err)
					return
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					err := fmt.Errorf("file watch event is not ok")
					gkill_log.Debug.Fatal(err)
					return
				}
				gkill_log.Debug.Print(err)
			}
		}
	}()

	return &watchTargetEntry{
		filename:           filename,
		rep:                rep,
		ignoreFilePrefixes: ignoreFilePrefixes,

		watcher:        watcher,
		requestCloseCh: requestCloseCh,

		watchUsers: map[string]struct{}{},
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
	// 消したあとにどのユーザも見ていなかったら消す。
	w.requestCloseCh <- struct{}{}
	return true, nil
}
