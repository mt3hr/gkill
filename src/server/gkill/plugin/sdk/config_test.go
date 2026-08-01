package sdk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestEnsureConfig_CreatesWhenMissing は config.json が無いとき既定設定で生成されることを確認する。
func TestEnsureConfig_CreatesWhenMissing(t *testing.T) {
	pluginDir := t.TempDir()
	defaults := Config{
		"_comment":    "説明",
		"source_dirs": []string{"~/Kyou/Export"},
	}

	cfg, err := EnsureConfig(pluginDir, defaults)
	if err != nil {
		t.Fatalf("EnsureConfig failed: %v", err)
	}
	if cfg["_comment"] != "説明" {
		t.Errorf("returned config = %v, want defaults", cfg)
	}

	// ファイルとして書き出され、読み戻せること
	configPath := filepath.Join(pluginDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config.json not created: %v", err)
	}
	var written map[string]any
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("config.json is not valid JSON: %v", err)
	}
	if written["_comment"] != "説明" {
		t.Errorf("written config = %v, want defaults", written)
	}
	dirs, ok := written["source_dirs"].([]any)
	if !ok || len(dirs) != 1 || dirs[0] != "~/Kyou/Export" {
		t.Errorf("written source_dirs = %v, want [~/Kyou/Export]", written["source_dirs"])
	}
}

// TestEnsureConfig_KeepsExisting は既存の config.json を上書きしないことを確認する。
// ユーザが手で書いた設定を起動のたびに壊さないための、いちばん大事な性質。
func TestEnsureConfig_KeepsExisting(t *testing.T) {
	pluginDir := t.TempDir()
	existing := `{"source_dirs":["D:/my/logs/**/*.jsonl"]}`
	configPath := filepath.Join(pluginDir, "config.json")
	if err := os.WriteFile(configPath, []byte(existing), 0600); err != nil {
		t.Fatalf("failed to write existing config: %v", err)
	}

	defaults := Config{"source_dirs": []string{"~/.claude/projects"}}
	cfg, err := EnsureConfig(pluginDir, defaults)
	if err != nil {
		t.Fatalf("EnsureConfig failed: %v", err)
	}

	dirs, ok := cfg["source_dirs"].([]any)
	if !ok || len(dirs) != 1 || dirs[0] != "D:/my/logs/**/*.jsonl" {
		t.Errorf("returned source_dirs = %v, want the existing value", cfg["source_dirs"])
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config.json: %v", err)
	}
	if string(data) != existing {
		t.Errorf("config.json was rewritten:\n got: %s\nwant: %s", data, existing)
	}
}

// TestEnsureConfig_NoDefaults は DefaultConfig が nil のときファイルを作らないことを確認する。
// 既定設定を持たない既存プラグインの挙動を変えないため。
func TestEnsureConfig_NoDefaults(t *testing.T) {
	pluginDir := t.TempDir()

	cfg, err := EnsureConfig(pluginDir, nil)
	if err != nil {
		t.Fatalf("EnsureConfig failed: %v", err)
	}
	if len(cfg) != 0 {
		t.Errorf("returned config = %v, want empty", cfg)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "config.json")); !os.IsNotExist(err) {
		t.Errorf("config.json should not be created, stat err = %v", err)
	}
}

// TestEnsureConfig_EmptyPluginDir は --gkill-plugin-dir 無しの手動起動で
// カレントディレクトリにファイルを作らないことを確認する。
func TestEnsureConfig_EmptyPluginDir(t *testing.T) {
	// カレントディレクトリを汚さないようテスト用の一時ディレクトリへ移る
	t.Chdir(t.TempDir())

	defaults := Config{"source_dirs": []string{"~/.claude/projects"}}
	cfg, err := EnsureConfig("", defaults)
	if err != nil {
		t.Fatalf("EnsureConfig failed: %v", err)
	}
	if len(cfg) != 0 {
		t.Errorf("returned config = %v, want empty", cfg)
	}
	if _, err := os.Stat("config.json"); !os.IsNotExist(err) {
		t.Errorf("config.json should not be created, stat err = %v", err)
	}
}
