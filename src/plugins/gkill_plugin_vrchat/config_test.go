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

func TestDefaultConfigRoundTrips(t *testing.T) {
	// --gkill-print-config が出すものがそのまま config.json として読めること
	data, err := json.MarshalIndent(defaultConfig(), "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pluginDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	config := configOf(pluginDir, nil)
	if len(config.Patterns) != 1 {
		t.Fatalf("Patterns = %v", config.Patterns)
	}
	if strings.HasPrefix(config.Patterns[0], "~") {
		t.Errorf("~ が展開されていない: %q", config.Patterns[0])
	}
	if !strings.Contains(strings.ReplaceAll(config.Patterns[0], `\`, "/"), "VRChat/VRChat") {
		t.Errorf("既定が VRChat の書き出し先を指していない: %q", config.Patterns[0])
	}
}

func TestConfigOfRereadsFile(t *testing.T) {
	// SDKはconfig.jsonをプロセス起動時に一度しか読まない。
	// 手編集や設定画面からの保存を再起動なしで反映するため毎回読み直す。
	pluginDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), []byte(`{"source_dirs":["C:/one"]}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	config := configOf(pluginDir, sdk.Config{configKeySourceDirs: []string{"C:/起動時のもの"}})
	if !slices.Contains(config.Patterns, "C:/one") {
		t.Errorf("ファイルの内容が使われていない: %v", config.Patterns)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), []byte(`{"source_dirs":["C:/two"]}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if config := configOf(pluginDir, nil); !slices.Contains(config.Patterns, "C:/two") {
		t.Errorf("書き換えが反映されていない: %v", config.Patterns)
	}
}

func TestConfigOfFallsBackToDefaultPatterns(t *testing.T) {
	pluginDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), []byte(`{"source_dirs":[]}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	config := configOf(pluginDir, nil)
	if len(config.Patterns) == 0 {
		t.Error("空指定のときに既定の場所へ倒れていない")
	}
}

func TestSplitSourceDirsForm(t *testing.T) {
	// 設定画面のテキストエリアは1行1指定。展開はせずに書いたとおり保存する
	got := splitSourceDirsForm("~/AppData/LocalLow/VRChat/VRChat\r\n\n  $HOME/Kyou/VRChat_*  \n")
	want := []string{"~/AppData/LocalLow/VRChat/VRChat", "$HOME/Kyou/VRChat_*"}
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
	if manifest.DataType != dataType {
		t.Errorf("data_type = %q, want %q", manifest.DataType, dataType)
	}
	if manifest.MinGkillVersion == "" {
		t.Error("min_gkill_version が空")
	}
	// 返す種類（TimeIs / URLog / Kmemo）とタグを全部宣言していないと、宣言していない種類は捨てられる
	for _, kind := range []string{"timeis", "urlog", "kmemo", "tag"} {
		if !slices.Contains(manifest.Provides, kind) {
			t.Errorf("provides に %q が無い: %v", kind, manifest.Provides)
		}
	}
}
