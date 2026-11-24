package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 既定の config.json は往復できて、既定の取り込み元を指す。
func TestDefaultConfigRoundTrip(t *testing.T) {
	encoded, err := json.Marshal(defaultConfig())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded sdk.Config
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	patterns := sdk.ParseSourcePatterns(decoded[configKeySourceDirs], "")
	if len(patterns) == 0 {
		t.Fatal("既定の source_dirs が空")
	}
	joined := strings.Join(patterns, "\n")
	if !strings.Contains(joined, filepath.Join(".gemini", "antigravity")) &&
		!strings.Contains(joined, ".gemini/antigravity") {
		t.Errorf("既定が ~/.gemini/antigravity を指していない: %v", patterns)
	}
	if _, ok := decoded[configKeyComment]; !ok {
		t.Error("書式の説明が無い")
	}
	if _, ok := decoded[configKeyExampleSourceDirs]; !ok {
		t.Error("記入例が無い")
	}
}

// configOf はハンドラのたびに config.json を読み直す（手で編集しても次の検索から効く）。
func TestConfigOfRereadsFile(t *testing.T) {
	pluginDir := t.TempDir()
	write := func(dirs []string) {
		t.Helper()
		data, err := json.Marshal(sdk.Config{configKeySourceDirs: dirs})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), data, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	write([]string{"C:/example/first"})
	if got := configOf(pluginDir, nil).Patterns; len(got) != 1 || got[0] != "C:/example/first" {
		t.Fatalf("1回目 = %v", got)
	}
	write([]string{"C:/example/second"})
	if got := configOf(pluginDir, nil).Patterns; len(got) != 1 || got[0] != "C:/example/second" {
		t.Errorf("読み直していない = %v", got)
	}
}

// source_dirs が空なら既定の場所へ倒す。
func TestConfigOfFallsBackToDefault(t *testing.T) {
	pluginDir := t.TempDir()
	data, err := json.Marshal(sdk.Config{configKeySourceDirs: []string{}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := configOf(pluginDir, nil).Patterns; len(got) != len(defaultSourcePatterns()) {
		t.Errorf("既定へ倒れていない = %v", got)
	}
}

// 設定画面のテキストエリア（1行1指定）を配列にする。展開はしない。
func TestSplitSourceDirsForm(t *testing.T) {
	got := splitSourceDirsForm("~/.gemini/antigravity\r\n\n C:/example/kyou ")
	if len(got) != 2 || got[0] != "~/.gemini/antigravity" || got[1] != "C:/example/kyou" {
		t.Errorf("splitSourceDirsForm = %q", got)
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
	// タグも型別データも出さないので provides は書かない
	// (タグを付けると rykv の既定の絞り込み「タグ無し」から漏れて何も表示されなくなる)
	if len(manifest.Provides) != 0 {
		t.Errorf("provides = %v, want 無し", manifest.Provides)
	}
}
