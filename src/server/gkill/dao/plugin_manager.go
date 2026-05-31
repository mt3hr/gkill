package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// PluginManager はユーザごとのプラグインの発見・ライフサイクルを管理する。
// $GKILL_HOME/plugins/{userID}/ 以下のサブディレクトリを走査し、
// manifest.json を持つものを PluginRepository として管理する。
type PluginManager struct {
	userID     string
	pluginsDir string // $GKILL_HOME/plugins/{userID}/
	plugins    []reps.PluginRepository
}

// newPluginManager はユーザ別の PluginManager を生成する。
// まだプラグインの発見は行わない。
func newPluginManager(userID string) *PluginManager {
	// GKILL_HOME は InitGkillOptions() で設定される確定済みパスを使う。
	// gkill_options.GkillHomeDir は "$HOME/gkill" のような未展開文字列のため、
	// Windowsで$HOMEが未設定だと空になることがある。
	pluginsBaseDir := os.ExpandEnv("$GKILL_HOME")
	if pluginsBaseDir == "" || pluginsBaseDir == "$GKILL_HOME" {
		pluginsBaseDir = filepath.Clean(os.ExpandEnv(gkill_options.GkillHomeDir))
	}
	pluginsDir := filepath.Join(pluginsBaseDir, "plugins", userID)
	return &PluginManager{
		userID:     userID,
		pluginsDir: pluginsDir,
	}
}

// DiscoverPlugins は pluginsDir 以下を走査してプラグインを発見・登録する。
// 配置パターン: {pluginsDir}/{pluginName}/ に manifest.json・実行ファイル・データを置く。
// 例: plugins/yamato/gkill_claude/manifest.json, gkill_claude.exe, conversations.json
// すでに登録済みのプラグインはスキップする（重複防止）。
// 発見失敗は警告ログに記録し、gkill本体の起動を止めない。
func (pm *PluginManager) DiscoverPlugins(ctx context.Context) error {
	if err := os.MkdirAll(pm.pluginsDir, os.ModePerm); err != nil {
		// ディレクトリ作成失敗はプラグイン無しとして扱う（警告のみ）
		slog.Warn(fmt.Sprintf("plugin dir create failed for user %s: %v", pm.userID, err))
		return nil
	}

	entries, err := os.ReadDir(pm.pluginsDir)
	if err != nil {
		slog.Warn(fmt.Sprintf("plugin dir read failed for user %s: %v", pm.userID, err))
		return nil
	}

	for _, entry := range entries {
		// サブディレクトリのみ対象（1プラグイン = 1ディレクトリ）
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(pm.pluginsDir, entry.Name())
		manifest, err := pm.loadManifest(pluginDir)
		if err != nil {
			// manifest.json がないディレクトリは無視
			continue
		}

		// 同名プラグインの重複チェック
		alreadyLoaded := false
		for _, existing := range pm.plugins {
			if existing.GetManifest().Name == manifest.Name {
				alreadyLoaded = true
				break
			}
		}
		if alreadyLoaded {
			slog.Info(fmt.Sprintf("plugin already loaded, skipping: %s", manifest.Name))
			continue
		}

		// プロトコルバージョン確認
		if manifest.ProtocolVersion != "1" {
			slog.Warn(fmt.Sprintf("plugin %s uses unsupported protocol version %s, skipping", manifest.Name, manifest.ProtocolVersion))
			continue
		}

		repo := reps.NewPluginRepository(pm.userID, pluginDir, *manifest)
		pm.plugins = append(pm.plugins, repo)
		slog.Info(fmt.Sprintf("plugin discovered: %s (user=%s, repName=%s)", manifest.Name, pm.userID, manifest.RepName))
	}

	return nil
}

// loadManifest は指定ディレクトリの manifest.json を読み込む。
func (pm *PluginManager) loadManifest(pluginDir string) (*gkill_plugin.PluginManifest, error) {
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest.json not found in %s: %w", pluginDir, err)
	}

	var manifest gkill_plugin.PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest.json in %s: %w", pluginDir, err)
	}

	if manifest.Name == "" {
		return nil, fmt.Errorf("manifest.json in %s: name is required", pluginDir)
	}
	if manifest.Executable == "" {
		return nil, fmt.Errorf("manifest.json in %s: executable is required", pluginDir)
	}
	if manifest.RepName == "" {
		return nil, fmt.Errorf("manifest.json in %s: rep_name is required", pluginDir)
	}
	if manifest.DataType == "" {
		return nil, fmt.Errorf("manifest.json in %s: data_type is required", pluginDir)
	}

	return &manifest, nil
}

// GetRepositories は発見済みのプラグインリポジトリ一覧を Repository スライスとして返す。
func (pm *PluginManager) GetRepositories() []reps.Repository {
	result := make([]reps.Repository, 0, len(pm.plugins))
	for _, p := range pm.plugins {
		result = append(result, p)
	}
	return result
}

// GetPluginRepositories は発見済みのプラグインリポジトリ一覧をPluginRepository スライスとして返す。
func (pm *PluginManager) GetPluginRepositories() []reps.PluginRepository {
	return pm.plugins
}

// GetPluginByName は名前でプラグインを検索する。見つからなければ nil を返す。
func (pm *PluginManager) GetPluginByName(name string) reps.PluginRepository {
	for _, p := range pm.plugins {
		if p.GetManifest().Name == name {
			return p
		}
	}
	return nil
}

// GetPluginByRepName はリポジトリ表示名でプラグインを検索する。見つからなければ nil を返す。
func (pm *PluginManager) GetPluginByRepName(repName string) reps.PluginRepository {
	for _, p := range pm.plugins {
		if p.GetManifest().RepName == repName {
			return p
		}
	}
	return nil
}

// CloseAll は全プラグインプロセスを終了する。gkillサーバのシャットダウン時に呼ぶ。
func (pm *PluginManager) CloseAll(ctx context.Context) error {
	var errs []error
	for _, p := range pm.plugins {
		if err := p.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error closing plugin %s: %w", p.GetManifest().Name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing plugins: %v", errs)
	}
	return nil
}
