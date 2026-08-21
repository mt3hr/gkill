package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// PluginManager はユーザごとのプラグインの発見・ライフサイクルを管理する。
// $GKILL_HOME/plugins/{userID}/ 以下のサブディレクトリを走査し、
// manifest.json を持つものを PluginRepository として管理する。
//
// 複数のHTTPハンドラから並行に触られるので、pluginsへのアクセスはmuで保護する。
type PluginManager struct {
	userID     string
	pluginsDir string // $GKILL_HOME/plugins/{userID}/

	mu      sync.RWMutex
	plugins []reps.PluginRepository
	// discovered は一度でも走査を済ませたかを表す。
	// 走査はディレクトリ列挙とmanifest.jsonの読み込みを伴うので、
	// リクエストのたびに繰り返さないためのフラグ。
	discovered bool
}

// isSingleSafePathElement は値を単一のパス要素として使ってよいか検証する。
// 区切り文字・親ディレクトリ参照・空文字を含むものを拒否する。
func isSingleSafePathElement(element string) bool {
	if element == "" || element == "." || element == ".." {
		return false
	}
	if strings.ContainsAny(element, `/\`) {
		return false
	}
	return filepath.Clean(element) == element
}

// newPluginManager はユーザ別の PluginManager を生成する。
// まだプラグインの発見は行わない。
// userID がパス要素として不正な場合はプラグイン無しとして扱う(pluginsDirを空にする)。
func newPluginManager(userID string) *PluginManager {
	// GKILL_HOME は InitGkillOptions() で設定される確定済みパスを使う。
	// gkill_options.GkillHomeDir は "$HOME/gkill" のような未展開文字列のため、
	// Windowsで$HOMEが未設定だと空になることがある。
	pluginsBaseDir := os.ExpandEnv("$GKILL_HOME")
	if pluginsBaseDir == "" || pluginsBaseDir == "$GKILL_HOME" {
		pluginsBaseDir = filepath.Clean(os.ExpandEnv(gkill_options.GkillHomeDir))
	}
	pluginsDir := ""
	if isSingleSafePathElement(userID) {
		pluginsDir = filepath.Join(pluginsBaseDir, "plugins", userID)
	} else {
		slog.Log(context.Background(), gkill_log.Warn, "invalid user id for plugin dir, plugins disabled", "user_id", fmt.Sprintf("%q", userID))
	}
	return &PluginManager{
		userID:     userID,
		pluginsDir: pluginsDir,
	}
}

// DiscoverPlugins は pluginsDir 以下を走査してプラグインを発見・登録する。
// 配置パターン: {pluginsDir}/{pluginName}/ に manifest.json・実行ファイル・データを置く。
// 例: plugins/mt3hr/gkill_claude/manifest.json, gkill_claude.exe, conversations.json
// すでに登録済みのプラグインはスキップする（重複防止）。
// 発見失敗は警告ログに記録し、gkill本体の起動を止めない。
func (pm *PluginManager) DiscoverPlugins(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.discoverPluginsLocked(ctx)
}

// EnsureDiscovered はまだ走査していなければ走査する。
//
// プラグインを参照するだけのハンドラから呼ぶことを想定している。
// DiscoverPlugins をリクエストのたびに呼ぶと、ディレクトリ列挙と
// manifest.json の読み込みが件数ぶん繰り返されるため分けてある。
func (pm *PluginManager) EnsureDiscovered(ctx context.Context) error {
	pm.mu.RLock()
	discovered := pm.discovered
	pm.mu.RUnlock()
	if discovered {
		return nil
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.discovered {
		return nil
	}
	return pm.discoverPluginsLocked(ctx)
}

// discoverPluginsLocked は pm.mu をロック済みの状態で走査を行う。
func (pm *PluginManager) discoverPluginsLocked(ctx context.Context) error {
	// 走査を試みた時点で以降のEnsureDiscoveredは走らせない。
	// プラグインが1つも無い構成で毎回ディレクトリを舐めないようにするため、
	// 失敗して途中で返る場合も含めてここで立てる。
	pm.discovered = true

	// pluginsDirが空 = userIDが不正でプラグイン無効
	if pm.pluginsDir == "" {
		return nil
	}
	if err := os.MkdirAll(pm.pluginsDir, os.ModePerm); err != nil {
		// ディレクトリ作成失敗はプラグイン無しとして扱う（警告のみ）
		slog.Log(ctx, gkill_log.Warn, "plugin dir create failed", "user_id", fmt.Sprintf("%q", pm.userID), "error", fmt.Sprintf("%q", err))
		return nil
	}

	entries, err := os.ReadDir(pm.pluginsDir)
	if err != nil {
		slog.Log(ctx, gkill_log.Warn, "plugin dir read failed", "user_id", fmt.Sprintf("%q", pm.userID), "error", fmt.Sprintf("%q", err))
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
			// 再走査のたびに出るので Debug にしている。
			// Info にすると再走査1回につきプラグイン数ぶんの行が積み上がる。
			slog.Log(ctx, gkill_log.Debug, "plugin already loaded, skipping", "plugin_name", fmt.Sprintf("%q", manifest.Name))
			continue
		}

		// プロトコルバージョン確認
		if manifest.ProtocolVersion != "1" {
			slog.Log(ctx, gkill_log.Warn, "plugin uses unsupported protocol version, skipping", "plugin_name", fmt.Sprintf("%q", manifest.Name), "protocol_version", fmt.Sprintf("%q", manifest.ProtocolVersion))
			continue
		}

		repo := reps.NewPluginRepository(pm.userID, pluginDir, *manifest)
		pm.plugins = append(pm.plugins, repo)
		slog.Log(ctx, gkill_log.Info, "plugin discovered", "plugin_name", fmt.Sprintf("%q", manifest.Name), "user_id", fmt.Sprintf("%q", pm.userID), "rep_name", fmt.Sprintf("%q", manifest.RepName))
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
	// executable は起動時に filepath.Join(pluginDir, ...) されるので、区切り文字や親参照を
	// 含むとプラグインフォルダ外のバイナリを起動できてしまう（サードパーティ配布への備え）。
	if !isSingleSafePathElement(manifest.Executable) {
		return nil, fmt.Errorf("manifest.json in %s: executable must be a single path element", pluginDir)
	}
	if manifest.RepName == "" {
		return nil, fmt.Errorf("manifest.json in %s: rep_name is required", pluginDir)
	}
	if manifest.DataType == "" {
		return nil, fmt.Errorf("manifest.json in %s: data_type is required", pluginDir)
	}

	// providesの検証。
	// 必須項目と違ってエラーにはせず「未知の値だけ落として警告」にする。
	// providesのタイプミスでプラグインごと無効化すると、
	// それまで出ていたタイムライン上の記録まで丸ごと消えてしまうため。
	validatedKinds := make([]gkill_plugin.PluginProvidedKind, 0, len(manifest.Provides))
	seenKinds := map[gkill_plugin.PluginProvidedKind]struct{}{}
	for _, kind := range manifest.Provides {
		if !slices.Contains(gkill_plugin.AllPluginProvidedKinds, kind) {
			slog.Log(context.Background(), gkill_log.Warn, "manifest.json has unknown provides, ignored", "plugin_dir", fmt.Sprintf("%q", pluginDir), "provides", fmt.Sprintf("%q", kind))
			continue
		}
		if _, duplicated := seenKinds[kind]; duplicated {
			continue
		}
		seenKinds[kind] = struct{}{}
		validatedKinds = append(validatedKinds, kind)
	}
	manifest.Provides = validatedKinds

	// data_typeとprovidesの食い違いを警告する。
	// クライアントはdata_typeの接頭辞で型別ビューを出し分けるので、
	// providesにkcを書いてもdata_typeが"kc"で始まらなければ
	// 型別データを取りに行く経路が一度も走らない。
	for _, kind := range validatedKinds {
		if !kind.IsTyped() {
			continue
		}
		if !strings.HasPrefix(manifest.DataType, string(kind)) {
			slog.Log(context.Background(), gkill_log.Warn, "manifest.json declares provides but data_type does not match; the client will never request the typed view", "plugin_dir", fmt.Sprintf("%q", pluginDir), "provides", fmt.Sprintf("%q", kind), "data_type", fmt.Sprintf("%q", manifest.DataType))
		}
	}

	return &manifest, nil
}

// GetRepositories は発見済みのプラグインリポジトリ一覧を Repository スライスとして返す。
func (pm *PluginManager) GetRepositories() []reps.Repository {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]reps.Repository, 0, len(pm.plugins))
	for _, p := range pm.plugins {
		result = append(result, p)
	}
	return result
}

// GetPluginRepositories は発見済みのプラグインリポジトリ一覧をPluginRepository スライスとして返す。
// 内部スライスをそのまま返すと再走査のappendと競合するのでコピーを返す。
func (pm *PluginManager) GetPluginRepositories() []reps.PluginRepository {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return append([]reps.PluginRepository(nil), pm.plugins...)
}

// GetPluginByName は名前でプラグインを検索する。見つからなければ nil を返す。
func (pm *PluginManager) GetPluginByName(name string) reps.PluginRepository {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, p := range pm.plugins {
		if p.GetManifest().Name == name {
			return p
		}
	}
	return nil
}

// GetPluginByRepName はリポジトリ表示名でプラグインを検索する。見つからなければ nil を返す。
func (pm *PluginManager) GetPluginByRepName(repName string) reps.PluginRepository {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, p := range pm.plugins {
		if p.GetManifest().RepName == repName {
			return p
		}
	}
	return nil
}

// CloseAll は全プラグインプロセスを終了する。gkillサーバのシャットダウン時に呼ぶ。
func (pm *PluginManager) CloseAll(ctx context.Context) error {
	plugins := pm.GetPluginRepositories()

	var errs []error
	for _, p := range plugins {
		if err := p.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error closing plugin %s: %w", p.GetManifest().Name, err))
		}
	}
	return errors.Join(errs...)
}
