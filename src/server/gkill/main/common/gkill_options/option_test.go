package gkill_options

import (
	"runtime"
	"testing"
	"time"
)

func TestDefaultGkillHomeDir(t *testing.T) {
	if GkillHomeDir == "" {
		t.Error("GkillHomeDir should not be empty")
	}
	if GkillHomeDir != "$HOME/gkill" {
		t.Errorf("GkillHomeDir = %q, want $HOME/gkill", GkillHomeDir)
	}
}

func TestDefaultConfigDir(t *testing.T) {
	if ConfigDir != "$HOME/gkill/configs" {
		t.Errorf("ConfigDir = %q, want $HOME/gkill/configs", ConfigDir)
	}
}

func TestDefaultCacheDir(t *testing.T) {
	if CacheDir != "$HOME/gkill/caches" {
		t.Errorf("CacheDir = %q, want $HOME/gkill/caches", CacheDir)
	}
}

func TestDefaultLogDir(t *testing.T) {
	if LogDir != "$HOME/gkill/logs" {
		t.Errorf("LogDir = %q, want $HOME/gkill/logs", LogDir)
	}
}

func TestDefaultIsCacheInMemory(t *testing.T) {
	if !IsCacheInMemory {
		t.Error("IsCacheInMemory should default to true")
	}
}

func TestDefaultDisableTLSForce(t *testing.T) {
	if DisableTLSForce {
		t.Error("DisableTLSForce should default to false")
	}
}

func TestDefaultGoroutinePool(t *testing.T) {
	if GoroutinePool != runtime.NumCPU() {
		t.Errorf("GoroutinePool = %d, want %d", GoroutinePool, runtime.NumCPU())
	}
}

func TestDefaultCacheClearCountLimit(t *testing.T) {
	if CacheClearCountLimit != 3000 {
		t.Errorf("CacheClearCountLimit = %d, want 3000", CacheClearCountLimit)
	}
}

func TestDefaultCacheUpdateDuration(t *testing.T) {
	if CacheUpdateDuration != 1*time.Minute {
		t.Errorf("CacheUpdateDuration = %v, want 1m", CacheUpdateDuration)
	}
}

func TestIDFIgnoreNotEmpty(t *testing.T) {
	if len(IDFIgnore) == 0 {
		t.Error("IDFIgnore should not be empty")
	}
}

func TestCachePointers(t *testing.T) {
	// All cache pointers should point to IsCacheInMemory
	ptrs := []*bool{
		CacheKmemoReps, CacheKCReps, CacheURLogReps, CacheNlogReps,
		CacheTimeIsReps, CacheMiReps, CacheLantanaReps, CacheIDFKyouReps,
		CacheTagReps, CacheTextReps, CacheNotificationReps, CacheReKyouReps,
		CacheGitCommitLogReps,
	}
	for i, p := range ptrs {
		if p == nil {
			t.Errorf("cache pointer [%d] is nil", i)
			continue
		}
		if p != &IsCacheInMemory {
			t.Errorf("cache pointer [%d] does not point to IsCacheInMemory", i)
		}
	}
}
