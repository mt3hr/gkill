package rep_cache_updater

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

const defaultDebounce = 300 * time.Millisecond

// --- global watcher hub (single watcher, single event loop) ---

var (
	globalOnce sync.Once
	globalHub  *watcherHub
	globalErr  error

	instanceSeq uint64
)

type watcherHub struct {
	w *fsnotify.Watcher

	m       sync.Mutex
	targets map[string]*watchTargetEntry // keySlash -> entry
	states  map[string]*targetState      // keySlash -> state

	debounce time.Duration
}

type targetState struct {
	timer    *time.Timer
	updating bool
	dirty    bool
}

func getHub() (*watcherHub, error) {
	globalOnce.Do(func() {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			globalErr = err
			return
		}
		h := &watcherHub{
			w:        w,
			targets:  map[string]*watchTargetEntry{},
			states:   map[string]*targetState{},
			debounce: defaultDebounce,
		}
		globalHub = h
		go h.runEventLoop()
	})
	return globalHub, globalErr
}

func normalizeKey(p string) string {
	return filepath.ToSlash(filepath.Clean(p))
}

func normalizeOSPath(p string) string {
	// filename may be slash-based; convert to OS style just in case.
	return filepath.Clean(filepath.FromSlash(p))
}

// --- public ctor ---

func NewFileRepCacheUpdater(skip *bool) (FileRepCacheUpdater, error) {
	h, err := getHub()
	if err != nil {
		return nil, err
	}
	id := atomic.AddUint64(&instanceSeq, 1)

	return &fileRepCacheUpdaterImpl{
		hub:     h,
		skip:    skip,
		instID:  id,
		entries: map[string]struct{}{}, // local bookkeeping: "keySlash|userID"
	}, nil
}

// --- updater instance (thin wrapper over global hub) ---

type fileRepCacheUpdaterImpl struct {
	m sync.Mutex

	hub    *watcherHub
	skip   *bool
	instID uint64

	// local bookkeeping (so Register double-call in same instance doesn't multiply owners)
	entries map[string]struct{} // "keySlash|userID"
}

func (f *fileRepCacheUpdaterImpl) ownerKey(userID string) string {
	// distinct per NewFileRepCacheUpdater() instance
	return fmt.Sprintf("%d|%s", f.instID, userID)
}

func (f *fileRepCacheUpdaterImpl) localKey(keySlash, userID string) string {
	return keySlash + "|" + userID
}

func (f *fileRepCacheUpdaterImpl) RegisterWatchFileRep(rep CacheUpdatable, filename string, ignoreFilePrefixes []string, userID string) error {
	key := normalizeKey(filename)

	f.m.Lock()
	lk := f.localKey(key, userID)
	if _, ok := f.entries[lk]; ok {
		f.m.Unlock()
		return nil
	}
	f.entries[lk] = struct{}{}
	f.m.Unlock()

	return f.hub.register(f.ownerKey(userID), rep, filename, ignoreFilePrefixes, f.skip)
}

func (f *fileRepCacheUpdaterImpl) RemoveWatchFileRep(filename string, userID string) error {
	key := normalizeKey(filename)

	f.m.Lock()
	lk := f.localKey(key, userID)
	if _, ok := f.entries[lk]; !ok {
		f.m.Unlock()
		return nil
	}
	delete(f.entries, lk)
	f.m.Unlock()

	return f.hub.unregister(f.ownerKey(userID), filename)
}

// Close は「このインスタンスが登録した分を解除する」だけ。
// グローバル watcher は Close しない（＝ private で運用）。
func (f *fileRepCacheUpdaterImpl) Close() error {
	f.m.Lock()
	keys := make([]string, 0, len(f.entries))
	for lk := range f.entries {
		keys = append(keys, lk)
	}
	f.entries = map[string]struct{}{}
	f.m.Unlock()

	// lk = "keySlash|userID"
	for _, lk := range keys {
		// split last '|'
		sep := -1
		for i := len(lk) - 1; i >= 0; i-- {
			if lk[i] == '|' {
				sep = i
				break
			}
		}
		if sep <= 0 {
			continue
		}
		keySlash := lk[:sep]
		userID := lk[sep+1:]

		// keySlash is normalized; but hub.unregister expects filename-ish.
		_ = f.hub.unregister(f.ownerKey(userID), keySlash)
	}

	return nil
}

// --- hub operations ---

func (h *watcherHub) register(ownerKey string, rep CacheUpdatable, filename string, ignoreFilePrefixes []string, skip *bool) error {
	key := normalizeKey(filename)
	osPath := normalizeOSPath(filename)

	h.m.Lock()
	defer h.m.Unlock()

	target, ok := h.targets[key]
	if !ok {
		// start watch
		if err := h.w.Add(osPath); err != nil {
			return fmt.Errorf("error at add watch file. filename = %s: %w", filename, err)
		}
		target = newWatchTargetEntry(filename, ignoreFilePrefixes)
		h.targets[key] = target
	}
	target.addOwner(ownerKey, rep, skip, ignoreFilePrefixes)

	if _, ok := h.states[key]; !ok {
		h.states[key] = &targetState{}
	}
	return nil
}

func (h *watcherHub) unregister(ownerKey string, filename string) error {
	key := normalizeKey(filename)

	h.m.Lock()
	defer h.m.Unlock()

	target, ok := h.targets[key]
	if !ok {
		return nil
	}

	empty := target.removeOwner(ownerKey)
	if !empty {
		return nil
	}

	// stop timer
	if st, ok := h.states[key]; ok && st != nil && st.timer != nil {
		st.timer.Stop()
	}
	delete(h.states, key)

	// stop watch
	_ = h.w.Remove(normalizeOSPath(target.filename))
	delete(h.targets, key)

	return nil
}

// --- single event loop ---

func (h *watcherHub) runEventLoop() {
	for {
		select {
		case event, ok := <-h.w.Events:
			if !ok {
				return
			}

			// Normalize routing key
			evKey := normalizeKey(event.Name)

			h.m.Lock()
			target := h.targets[evKey]
			h.m.Unlock()
			if target == nil {
				continue
			}

			// ignore paths
			if target.shouldIgnore(event.Name) {
				continue
			}

			// meaningful ops only
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}

			// if all active owners are skipping, do nothing
			if target.shouldSkipAll() {
				continue
			}

			h.scheduleUpdate(evKey)

		case err, ok := <-h.w.Errors:
			if !ok {
				return
			}
			gkill_log.Debug.Printf("fsnotify error: %v\n", err)
		}
	}
}

func (h *watcherHub) scheduleUpdate(key string) {
	h.m.Lock()
	defer h.m.Unlock()

	target := h.targets[key]
	if target == nil {
		return
	}

	st, ok := h.states[key]
	if !ok {
		st = &targetState{}
		h.states[key] = st
	}

	// If currently updating, just mark dirty (coalesce).
	if st.updating {
		st.dirty = true
		return
	}

	if st.timer != nil {
		st.timer.Reset(h.debounce)
		return
	}

	st.timer = time.AfterFunc(h.debounce, func() {
		h.triggerUpdate(key)
	})
}

func (h *watcherHub) triggerUpdate(key string) {
	go func() {
		h.runUpdateCoalesced(key)
	}()
}

func (h *watcherHub) runUpdateCoalesced(key string) {
	for {
		// snapshot
		h.m.Lock()
		target := h.targets[key]
		st := h.states[key]
		if target == nil || st == nil {
			h.m.Unlock()
			return
		}

		// timer fired
		st.timer = nil

		if st.updating {
			st.dirty = true
			h.m.Unlock()
			return
		}
		st.updating = true
		st.dirty = false

		reps := target.repsToUpdate() // already filtered by skip
		filename := target.filename
		h.m.Unlock()

		// execute UpdateCache outside lock
		for _, rep := range reps {
			if rep == nil {
				continue
			}
			if err := rep.UpdateCache(context.TODO()); err != nil {
				gkill_log.Debug.Printf("error at update cache. filename = %s: %v\n", filename, err)
				// 継続するかは好み。ここは「他repは続ける」運用にしてる
			}
		}

		// finish & maybe rerun once
		h.m.Lock()
		st = h.states[key]
		if st == nil {
			h.m.Unlock()
			return
		}
		st.updating = false
		again := st.dirty
		st.dirty = false
		h.m.Unlock()

		if !again {
			return
		}
	}
}
