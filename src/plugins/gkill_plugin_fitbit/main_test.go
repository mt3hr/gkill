package main

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// setPluginLogWriterForTest は stderr 側のログをバッファへ向け、終了時に戻す。
func setPluginLogWriterForTest(t *testing.T, w io.Writer) {
	t.Helper()
	sdk.SetLogWriter(w)
	t.Cleanup(func() { sdk.SetLogWriter(nil) })
}

// PostConfig の secondary_data_sources は、空欄を「全部合算する」の明示として空配列で保存する
// （既定へ戻すのではない。既定へ戻したければキーごと消す）。
func TestPostConfigSecondaryDataSourcesEmptyMeansSumAll(t *testing.T) {
	h := newHandler(t.TempDir())

	cfg, err := h.PostConfig(context.Background(), map[string]string{configKeySecondaryDataSources: ""}, sdk.Config{})
	if err != nil {
		t.Fatalf("PostConfig: %v", err)
	}
	saved, present := cfg[configKeySecondaryDataSources]
	if !present {
		t.Fatal("空欄なのにキーが保存されていない（既定へ戻ってしまう）")
	}
	if list, ok := saved.([]string); !ok || len(list) != 0 {
		t.Errorf("空欄 = %#v, want 空配列", saved)
	}

	cfg, err = h.PostConfig(context.Background(), map[string]string{configKeySecondaryDataSources: "Phone Health Connect\r\n\nGoogle Health App\n"}, sdk.Config{})
	if err != nil {
		t.Fatalf("PostConfig: %v", err)
	}
	if got := cfg[configKeySecondaryDataSources]; !reflect.DeepEqual(got, []string{"Phone Health Connect", "Google Health App"}) {
		t.Errorf("複数行 = %#v, want 2件", got)
	}

	// フォームにキーが無ければ触らない（他の欄だけ保存したとき）
	cfg, err = h.PostConfig(context.Background(), map[string]string{}, sdk.Config{configKeySecondaryDataSources: []string{"keep"}})
	if err != nil {
		t.Fatalf("PostConfig: %v", err)
	}
	if got := cfg[configKeySecondaryDataSources]; !reflect.DeepEqual(got, []string{"keep"}) {
		t.Errorf("キー無しで書き換わった: %#v", got)
	}
}

// 単独モード（gkill_server generate_plugin_cache）の BuildCache は、常駐ビルダを起こさず同期で1周し、
// 戻った時点でキャッシュが引けること。
func TestBuildCacheBuildsSynchronously(t *testing.T) {
	sourceDir := stepsSourceDir(t, stepsCSV(
		[3]string{"2025-12-15T02:00:00Z", "100", "Pixel Watch 2"},
		[3]string{"2025-12-15T03:00:00Z", "200", "Pixel Watch 2"},
	))
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	original := globalCache
	globalCache = c
	t.Cleanup(func() { globalCache = original })
	var logs bytes.Buffer
	setPluginLogWriterForTest(t, &logs)

	cfg := sdk.Config{configKeySourceDirs: []string{sourceDir}, configKeyTimezone: "Asia/Tokyo"}
	if err := newHandler(pluginDir).BuildCache(context.Background(), cfg); err != nil {
		t.Fatalf("BuildCache: %v", err)
	}
	metrics, err := c.QueryDailyMetrics(pluginDir, configOf(pluginDir, cfg), nil, nil, 0)
	if err != nil {
		t.Fatalf("QueryDailyMetrics: %v", err)
	}
	found := false
	for _, metric := range metrics {
		if metric.MetricKey == "steps_daily" && metric.DateLocal == "2025-12-15" && metric.NumValue == "300" {
			found = true
		}
	}
	if !found {
		t.Errorf("BuildCache から戻った時点で日次の値が引けない: %+v", metrics)
	}
	if bytes.Contains(logs.Bytes(), []byte("ERROR:")) {
		t.Errorf("同期構築で ERROR 行が出ている: %q", logs.String())
	}
}
