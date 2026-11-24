package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// buildOnce は構築が失敗したら cache_meta（build_state / build_error）に残し、stderr に ERROR 行を出して
// エラーを返す。常駐ビルダと単独モード（generate_plugin_cache）の両方がここを通るので、
// どちらで失敗しても設定画面の見え方は同じになる。成功なら build_state を上書きしない。
func TestBuildOnceRecordsFailureInCacheMeta(t *testing.T) {
	c, pluginDir := newTestCache(t)
	if err := c.openDB(pluginDir); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	originalCache, originalBuild := globalCache, buildCacheFn
	globalCache = c
	t.Cleanup(func() {
		globalCache = originalCache
		buildCacheFn = originalBuild
	})
	var logs bytes.Buffer
	sdk.SetLogWriter(&logs)
	t.Cleanup(func() { sdk.SetLogWriter(nil) })

	buildCacheFn = func(string, []string) error { return errors.New("boom") }
	if err := buildOnce(pluginDir, nil); err == nil || err.Error() != "boom" {
		t.Fatalf("buildOnce error = %v, want boom", err)
	}
	if got := c.getMeta("build_state"); got != "error" {
		t.Errorf("build_state = %q, want error", got)
	}
	if got := c.getMeta("build_error"); got != "boom" {
		t.Errorf("build_error = %q, want boom", got)
	}
	if !strings.Contains(logs.String(), "ERROR: "+appName+": build error: boom") {
		t.Errorf("ERROR 行が無い: %q", logs.String())
	}

	logs.Reset()
	buildCacheFn = func(string, []string) error {
		c.setMeta("build_state", "idle")
		return nil
	}
	if err := buildOnce(pluginDir, nil); err != nil {
		t.Fatalf("buildOnce: %v", err)
	}
	if got := c.getMeta("build_state"); got != "idle" {
		t.Errorf("成功後の build_state = %q, want idle（buildOnce は成功時に触らない）", got)
	}
	if strings.Contains(logs.String(), "ERROR:") {
		t.Errorf("成功なのに ERROR 行が出ている: %q", logs.String())
	}
}
