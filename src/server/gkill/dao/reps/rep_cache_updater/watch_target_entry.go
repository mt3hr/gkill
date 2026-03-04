package rep_cache_updater

import (
	"path/filepath"
	"reflect"
	"strings"
)

type ownerInfo struct {
	rep  CacheUpdatable
	skip *bool
}

type watchTargetEntry struct {
	filename string // original
	keySlash string // normalized key

	// union of ignore prefixes
	ignorePrefixes map[string]struct{}

	// ownerKey -> info
	owners map[string]ownerInfo
}

func newWatchTargetEntry(filename string, ignoreFilePrefixes []string) *watchTargetEntry {
	ip := map[string]struct{}{}
	for _, p := range ignoreFilePrefixes {
		ip[p] = struct{}{}
	}
	return &watchTargetEntry{
		filename:       filename,
		keySlash:       filepath.ToSlash(filepath.Clean(filename)),
		ignorePrefixes: ip,
		owners:         map[string]ownerInfo{},
	}
}

func (w *watchTargetEntry) addOwner(ownerKey string, rep CacheUpdatable, skip *bool, ignoreFilePrefixes []string) {
	w.owners[ownerKey] = ownerInfo{rep: rep, skip: skip}
	for _, p := range ignoreFilePrefixes {
		w.ignorePrefixes[p] = struct{}{}
	}
}

func (w *watchTargetEntry) removeOwner(ownerKey string) (empty bool) {
	delete(w.owners, ownerKey)
	return len(w.owners) == 0
}

func (w *watchTargetEntry) shouldIgnore(eventName string) bool {
	ev := filepath.ToSlash(filepath.Clean(eventName))
	for p := range w.ignorePrefixes {
		if strings.HasPrefix(ev, p) {
			return true
		}
	}
	return false
}

func (w *watchTargetEntry) shouldSkipAll() bool {
	if len(w.owners) == 0 {
		return true
	}
	for _, info := range w.owners {
		if info.skip == nil {
			return false
		}
		if !*info.skip {
			return false
		}
	}
	return true
}

func (w *watchTargetEntry) repsToUpdate() []CacheUpdatable {
	// Filter by skip=false and dedupe by pointer
	seen := map[uintptr]struct{}{}
	out := []CacheUpdatable{}

	for _, info := range w.owners {
		if info.rep == nil {
			continue
		}
		if info.skip != nil && *info.skip {
			continue
		}
		// dedupe
		v := reflect.ValueOf(info.rep)
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		var ptr uintptr
		if v.IsValid() && v.Kind() == reflect.Pointer {
			ptr = v.Pointer()
		}
		if ptr != 0 {
			if _, ok := seen[ptr]; ok {
				continue
			}
			seen[ptr] = struct{}{}
		}
		out = append(out, info.rep)
	}
	return out
}
