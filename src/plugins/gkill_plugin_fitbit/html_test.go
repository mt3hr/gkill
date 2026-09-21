package main

import (
	"strings"
	"testing"
)

// 設定画面は secondary_data_sources の現在値を textarea に出し、保存スクリプトが同じキーで
// postMessage に載せること（片方だけ欠けると、設定したつもりの値が保存されない）。
func TestRenderConfigHTMLShowsAndSavesSecondaryDataSources(t *testing.T) {
	config := pluginConfig{
		Patterns:             []string{"~/Kyou/GoogleTakeout_*"},
		Timezone:             "Asia/Tokyo",
		SecondaryDataSources: []string{"Phone Health Connect", "Google <Health> App"},
	}
	html := renderConfigHTML("C:/plugins/x", config, cacheStats{})
	for _, want := range []string{
		`id="gkill_secondary_data_sources"`,
		"Phone Health Connect\nGoogle &lt;Health&gt; App", // 現在値（HTML エスケープ済み）
		"secondary_data_sources: secondaryDataSources",    // 保存スクリプトのキー
		`id="gkill_source_dirs"`,
		"gkill_plugin_config",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("設定画面に %q が無い", want)
		}
	}
	if strings.Contains(html, "Google <Health> App") {
		t.Error("設定値が HTML エスケープされずに出ている")
	}
}
