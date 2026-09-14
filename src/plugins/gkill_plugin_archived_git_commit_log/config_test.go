package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 既定設定が JSON を往復しても読めること（配置スクリプトが --gkill-print-config の出力をそのまま置く）。
func TestDefaultConfigRoundTrips(t *testing.T) {
	data, err := json.Marshal(defaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	var cfg sdk.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	pluginDir := t.TempDir()
	config := configOf(pluginDir, cfg)
	if len(config.Patterns) != 0 {
		t.Errorf("既定の source_dirs は空のはず: %v", config.Patterns)
	}
	if config.MaxGitDirBytes != defaultMaxGitDirMB<<20 {
		t.Errorf("MaxGitDirBytes = %d, want %d MB", config.MaxGitDirBytes, defaultMaxGitDirMB)
	}
}

// config.json を書き換えたら次の呼び出しで反映されること（プロセスの再起動は不要）。
func TestConfigOfRereadsFile(t *testing.T) {
	pluginDir := t.TempDir()
	if err := sdk.SaveConfig(pluginDir, sdk.Config{
		configKeySourceDirs:  []string{"D:/backup/repos/*.zip", "~/Kyou/Box_A"},
		configKeyMaxGitDirMB: 8,
	}); err != nil {
		t.Fatal(err)
	}
	config := configOf(pluginDir, sdk.Config{configKeySourceDirs: "stale"})
	if len(config.Patterns) != 2 || config.Patterns[0] != "D:/backup/repos/*.zip" {
		t.Errorf("Patterns = %v", config.Patterns)
	}
	if config.MaxGitDirBytes != 8<<20 {
		t.Errorf("MaxGitDirBytes = %d, want 8MB", config.MaxGitDirBytes)
	}

	// 読めない config.json なら起動時の設定にフォールバック
	if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	config = configOf(pluginDir, sdk.Config{configKeySourceDirs: "fallback.zip"})
	if len(config.Patterns) != 1 || config.Patterns[0] != "fallback.zip" {
		t.Errorf("フォールバックしていない: %v", config.Patterns)
	}
}

func TestParsePositiveInt(t *testing.T) {
	if parsePositiveInt(float64(12)) != 12 || parsePositiveInt(3) != 3 || parsePositiveInt(json.Number("7")) != 7 {
		t.Error("数値が読めない")
	}
	if parsePositiveInt(float64(0)) != 0 || parsePositiveInt(-1) != 0 || parsePositiveInt("x") != 0 || parsePositiveInt(nil) != 0 {
		t.Error("0以下・非数値は 0 に倒す")
	}
}

func TestSplitSourceDirsForm(t *testing.T) {
	got := splitSourceDirsForm("D:/repos/a.zip\r\n\n  ~/Kyou/Box_A  \n")
	want := []string{"D:/repos/a.zip", "~/Kyou/Box_A"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(splitSourceDirsForm("")) != 0 {
		t.Error("空文字から要素が出ている")
	}
}

func TestManifestMatchesConstants(t *testing.T) {
	// ディレクトリ名 / name / executable は全部同じでなければならない。
	// Termux 側が pkill -KILL -f gkill_plugin_ で落とすので接頭辞も必須。
	var manifest struct {
		ProtocolVersion string   `json:"protocol_version"`
		Name            string   `json:"name"`
		DataType        string   `json:"data_type"`
		RepName         string   `json:"rep_name"`
		Executable      string   `json:"executable"`
		MinGkillVersion string   `json:"min_gkill_version"`
		Provides        []string `json:"provides"`
	}
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		t.Fatalf("manifest.json が壊れている: %v", err)
	}
	if manifest.ProtocolVersion != "1" {
		t.Errorf("protocol_version = %q", manifest.ProtocolVersion)
	}
	if manifest.Name != appName || manifest.Executable != appName {
		t.Errorf("name=%q executable=%q, want %q", manifest.Name, manifest.Executable, appName)
	}
	if !strings.HasPrefix(manifest.Name, "gkill_plugin_") {
		t.Errorf("name に gkill_plugin_ の接頭辞が無い: %q", manifest.Name)
	}
	if manifest.RepName != repName {
		t.Errorf("rep_name = %q, want %q", manifest.RepName, repName)
	}
	// data_type は native と同じ git_commit_log。別名にするとクライアントが型別ビューを出さず、
	// 稼働中リポジトリと重なるコミットが2件並ぶ。
	if manifest.DataType != dataType {
		t.Errorf("data_type = %q, want %q", manifest.DataType, dataType)
	}
	if dataType != "git_commit_log" {
		t.Errorf("dataType = %q, want git_commit_log", dataType)
	}
	if !slices.Equal(manifest.Provides, []string{"git_commit_log"}) {
		t.Errorf("provides = %v, want [git_commit_log]（型別アダプタが無いと native の経路に載らない）", manifest.Provides)
	}
	if manifest.MinGkillVersion == "" {
		t.Error("min_gkill_version が空")
	}
}
