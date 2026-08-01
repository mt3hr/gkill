package sdk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadConfig は --gkill-plugin-dir 引数で指定されたディレクトリの config.json を読み込む。
// ファイルが存在しない場合は空のConfigを返す（エラーにしない）。
func LoadConfig(pluginDir string) (Config, error) {
	configPath := filepath.Join(pluginDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return nil, fmt.Errorf("error at read config.json: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error at parse config.json: %w", err)
	}
	return cfg, nil
}

// EnsureConfig は pluginDir の config.json を読み込む。
// ファイルが無ければ defaults で作成してから返す。既存ファイルは決して上書きしない。
//
// プラグインの設定は manifest.json と同じフォルダの config.json を手で編集して行うが、
// ファイル自体が無いとキー名も書式も分からない。そこで起動時に雛形を置いておく。
//
// pluginDir が空(--gkill-plugin-dir 無しの手動起動)、または defaults が nil のときは
// 何も書かずに LoadConfig と同じ結果を返す。
func EnsureConfig(pluginDir string, defaults Config) (Config, error) {
	if pluginDir == "" || defaults == nil {
		return LoadConfig(pluginDir)
	}

	configPath := filepath.Join(pluginDir, "config.json")
	if _, err := os.Stat(configPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error at stat config.json: %w", err)
		}
		if err := SaveConfig(pluginDir, defaults); err != nil {
			return nil, err
		}
		return defaults, nil
	}

	return LoadConfig(pluginDir)
}

// SaveConfig は設定をpluginDirのconfig.jsonに保存する。
func SaveConfig(pluginDir string, cfg Config) error {
	if err := os.MkdirAll(pluginDir, os.ModePerm); err != nil {
		return fmt.Errorf("error at mkdir etc dir: %w", err)
	}
	configPath := filepath.Join(pluginDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error at marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("error at write config.json: %w", err)
	}
	return nil
}
