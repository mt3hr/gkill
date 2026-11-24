package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// manifest.json とコードの定数がずれていないことを見る。
//
// ずれても目の前ではエラーにならない: rep 名がずれると rep 名の絞り込みから漏れ、
// data_type がずれると型別の振り分けが合わなくなる。name / executable は
// 「ディレクトリ名・name・executable・配置先のフォルダ名が同じ文字列」という
// 配置の規約（src/plugins/README.md）そのもの。
func TestManifestMatchesCode(t *testing.T) {
	var manifest struct {
		ProtocolVersion string   `json:"protocol_version"`
		Name            string   `json:"name"`
		Version         string   `json:"version"`
		Description     string   `json:"description"`
		DataType        string   `json:"data_type"`
		RepName         string   `json:"rep_name"`
		Executable      string   `json:"executable"`
		MinGkillVersion string   `json:"min_gkill_version"`
		Provides        []string `json:"provides"`
	}
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		t.Fatalf("manifest.json が読めない: %v", err)
	}

	if manifest.Name != appName {
		t.Errorf("name = %q, want %q", manifest.Name, appName)
	}
	if manifest.Executable != appName {
		t.Errorf("executable = %q, want %q", manifest.Executable, appName)
	}
	if manifest.RepName != repName {
		t.Errorf("rep_name = %q, want %q", manifest.RepName, repName)
	}
	if manifest.DataType != dataType {
		t.Errorf("data_type = %q, want %q", manifest.DataType, dataType)
	}
	if manifest.ProtocolVersion != "1" {
		t.Errorf("protocol_version = %q, want 1（上げると古い gkill が読み込めない）", manifest.ProtocolVersion)
	}
	if manifest.MinGkillVersion == "" || manifest.Version == "" || manifest.Description == "" {
		t.Errorf("必須の欄が空: %+v", manifest)
	}
	// 型別データを返さないので provides は書かない（書くと型別索引を作ろうとする）。
	if len(manifest.Provides) != 0 {
		t.Errorf("provides = %v, want 空（このプラグインは型別データを返さない）", manifest.Provides)
	}
	if !strings.HasPrefix(manifest.Name, "gkill_plugin_") {
		t.Errorf("name = %q, want gkill_plugin_ 接頭辞（更新時にプロセスをまとめて止められなくなる）", manifest.Name)
	}
}

// 既定の config.json は Takeout を読む他の2本（fitbit / 位置情報）と同じ置き場を指す。
func TestDefaultConfigPointsAtTakeoutFolder(t *testing.T) {
	cfg := defaultConfig()
	dirs, ok := cfg[configKeySourceDirs].([]string)
	if !ok || len(dirs) != 1 || dirs[0] != defaultSourcePattern {
		t.Errorf("source_dirs = %v, want [%s]", cfg[configKeySourceDirs], defaultSourcePattern)
	}
	// --gkill-print-config がそのまま JSON にできること。
	if _, err := json.MarshalIndent(cfg, "", "  "); err != nil {
		t.Errorf("既定の config.json が JSON にできない: %v", err)
	}
}
