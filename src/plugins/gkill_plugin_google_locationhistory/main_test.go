package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 単独モード（gkill_server generate_plugin_cache）の BuildCache は、バックグラウンドの走査を起こさず
// 同期で1周し、戻った時点で点が引けること。
func TestBuildCacheRefreshesSynchronously(t *testing.T) {
	sourceDir := copyTestData(t, "timeline_edits_small.json", "gps_location_2024-04-18.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	original := globalCache
	globalCache = c
	t.Cleanup(func() { globalCache = original })
	var logs bytes.Buffer
	sdk.SetLogWriter(&logs)
	t.Cleanup(func() { sdk.SetLogWriter(nil) })

	cfg := sdk.Config{configKeySourceDirs: []string{sourceDir}}
	if err := newHandler(pluginDir).BuildCache(context.Background(), cfg); err != nil {
		t.Fatalf("BuildCache: %v", err)
	}
	if got := c.Stats(pluginDir, configOf(pluginDir, cfg)).TotalPoints; got == 0 {
		t.Error("BuildCache から戻った時点で1点も引けない")
	}
	if c.refreshing.Load() {
		t.Error("同期の構築なのにバックグラウンドの走査が起きている")
	}
	if bytes.Contains(logs.Bytes(), []byte("ERROR:")) {
		t.Errorf("同期構築で ERROR 行が出ている: %q", logs.String())
	}
}

// kickRefresh は走査の失敗を ERROR 行（gkill 本体の last_error が読む stderr）へ残す。
// 待たずに戻るので、refreshWG で終わりを待ってから見る。
func TestKickRefreshLogsRefreshFailure(t *testing.T) {
	var logs bytes.Buffer
	sdk.SetLogWriter(&logs)
	t.Cleanup(func() { sdk.SetLogWriter(nil) })

	// 通常ファイルの下を pluginDir にすると cache.db を作れず、refresh が失敗する
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &cache{}
	c.kickRefresh(filepath.Join(blocker, "plugin"), testConfig(t.TempDir()))
	c.refreshWG.Wait()
	if !bytes.Contains(logs.Bytes(), []byte("ERROR: "+appName+": refresh:")) {
		t.Errorf("走査の失敗が ERROR 行に残っていない: %q", logs.String())
	}
	if c.refreshing.Load() {
		t.Error("走査が終わったのに refreshing が立ったまま（次の kick が永久に無視される）")
	}
}
