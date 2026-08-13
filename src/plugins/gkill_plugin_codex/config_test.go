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
	var decoded sdk.Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	pluginDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	config := configOf(pluginDir, nil)
	if config.SubagentMode != subagentModeFold {
		t.Errorf("SubagentMode = %q", config.SubagentMode)
	}
	if config.ScanWorkers != 0 {
		t.Errorf("ScanWorkers = %d", config.ScanWorkers)
	}
	if len(config.Patterns) != 2 {
		t.Fatalf("Patterns = %v", config.Patterns)
	}
	for _, pattern := range config.Patterns {
		if strings.HasPrefix(pattern, "~") {
			t.Errorf("~ が展開されていない: %q", pattern)
		}
	}
	if !strings.Contains(config.Patterns[1], sessionIndexFileName) {
		t.Errorf("スレッド名の索引が既定に入っていない: %v", config.Patterns)
	}
}

func TestConfigOfRereadsFile(t *testing.T) {
	// SDKはconfig.jsonをプロセス起動時に一度しか読まない。
	// 手編集や設定画面からの保存を再起動なしで反映するため毎回読み直す。
	pluginDir := t.TempDir()
	first := `{"source_dirs":["C:/one"],"subagent_mode":"own_kyou","scan_workers":3}`
	if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), []byte(first), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	config := configOf(pluginDir, sdk.Config{configKeySourceDirs: []string{"C:/起動時のもの"}})
	if !slices.Contains(config.Patterns, "C:/one") {
		t.Errorf("ファイルの内容が使われていない: %v", config.Patterns)
	}
	if config.SubagentMode != subagentModeOwnKyou {
		t.Errorf("SubagentMode = %q", config.SubagentMode)
	}
	if config.ScanWorkers != 3 {
		t.Errorf("ScanWorkers = %d", config.ScanWorkers)
	}

	second := `{"source_dirs":["C:/two"]}`
	if err := os.WriteFile(filepath.Join(pluginDir, "config.json"), []byte(second), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	config = configOf(pluginDir, nil)
	if !slices.Contains(config.Patterns, "C:/two") {
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
	for _, pattern := range config.Patterns {
		if !strings.Contains(strings.ReplaceAll(pattern, `\`, "/"), ".codex") {
			t.Errorf("既定が ~/.codex を指していない: %q", pattern)
		}
	}
}

func TestParseSubagentMode(t *testing.T) {
	if got := parseSubagentMode("own_kyou"); got != subagentModeOwnKyou {
		t.Errorf("got %q", got)
	}
	// 知らない値は既定へ倒す
	for _, input := range []any{"", "fold", "しらない値", nil, 42} {
		if got := parseSubagentMode(input); got != subagentModeFold {
			t.Errorf("parseSubagentMode(%v) = %q, want %q", input, got, subagentModeFold)
		}
	}
}

func TestParseWorkers(t *testing.T) {
	cases := []struct {
		input any
		want  int
	}{
		{float64(4), 4}, {float64(0), 0}, {float64(-1), 0},
		{int(2), 2}, {nil, 0}, {"3", 0},
	}
	for _, c := range cases {
		if got := parseWorkers(c.input); got != c.want {
			t.Errorf("parseWorkers(%v) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestSplitSourceDirsForm(t *testing.T) {
	// 設定画面のテキストエリアは1行1指定。展開はせずに書いたとおり保存する
	got := splitSourceDirsForm("~/.codex/sessions\r\n\n  C:/backup/Codex_*  \n")
	want := []string{"~/.codex/sessions", "C:/backup/Codex_*"}
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
		ProtocolVersion string `json:"protocol_version"`
		Name            string `json:"name"`
		DataType        string `json:"data_type"`
		RepName         string `json:"rep_name"`
		Executable      string `json:"executable"`
		MinGkillVersion string `json:"min_gkill_version"`
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
}
