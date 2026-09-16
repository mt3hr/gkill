package common

// 編集前に読む: .claude/skills/gkill-cli-ops/SKILL.md（この領域の不変条件の正本）

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/spf13/cobra"
)

// generate_plugin_cache はプラグインのキャッシュ（$GKILL_HOME/caches/plugin_cache/{user}/{plugin}/cache.db）を
// 稼働中サーバ無しで同期構築する。プラグインバイナリを --gkill-build-cache 付きで単独起動し、
// 構築が終わるまで待つ（--gkill-print-manifest と同じ「stdio ループに入らない起動」の型）。
// 経緯と却下案: documents/adr/1003-generate-plugin-cache-runs-plugin-standalone.md

// pluginBuildResult はプラグインが stdout に1行で返す結果。
// 綴りは SDK（plugin/sdk/sdk.go の BuildCacheResult*）と一致させる。本体から SDK を import しないのは
// sdk/cache_path.go と同じ理由（プラグイン側ライブラリを本体へ持ち込まない）。
type pluginBuildResult string

const (
	// pluginBuildResultBuilt はキャッシュを構築した。
	pluginBuildResultBuilt pluginBuildResult = "built"
	// pluginBuildResultNoCache はキャッシュを持たないプラグイン（Handler.BuildCache が nil）。成功扱い。
	pluginBuildResultNoCache pluginBuildResult = "no_cache"
)

// generatePluginCacheAllPlugins は第1引数で「そのユーザーの全プラグイン」を表す語。
const generatePluginCacheAllPlugins = "all"

var GeneratePluginCacheCmd = &cobra.Command{
	Use:           "generate_plugin_cache",
	Short:         `generate_plugin_cache <plugin_name|all> <user_id...>`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 第1引数: プラグイン名(manifest.json の name = フォルダ名)か all、第2引数以降: 対象ユーザー。
		// clear_cache <mode> <user_id...> と同じ並び。
		if len(args) < 2 {
			return cmd.Usage()
		}
		pluginName := args[0]
		targetUserIDs := args[1:]

		// configs の DB を開くだけ。利用者の rep は読まない（GetRepositories を呼ばない）。
		if err := InitGkillServerAPI(); err != nil {
			return fmt.Errorf("error at init gkill server api: %w", err)
		}

		// 1ユーザが失敗しても残りは続け、最後にまとめて返す(1件でも失敗すれば exit 1)。
		var errs []error
		for _, targetUserID := range targetUserIDs {
			if err := GeneratePluginCache(cmd.Context(), targetUserID, pluginName, os.Stdout); err != nil {
				errs = append(errs, fmt.Errorf("error at generate plugin cache user id = %s: %w", targetUserID, err))
			}
		}
		return errors.Join(errs...)
	},
}

// pluginBuildTarget は単独モードで1回起動するプラグイン。
type pluginBuildTarget struct {
	Name            string
	ExecPath        string
	PluginDir       string
	ProtocolVersion string
}

// pluginCacheSource は利用者の実在確認とプラグイン一覧の取得。テストで差し替える。
type pluginCacheSource interface {
	AccountExists(ctx context.Context, userID string) (bool, error)
	PluginRepositories(ctx context.Context, userID string) ([]reps.PluginRepository, error)
}

// gkillPluginCacheSource は InitGkillServerAPI 済みの本番実装。
type gkillPluginCacheSource struct{}

func (gkillPluginCacheSource) AccountExists(ctx context.Context, userID string) (bool, error) {
	account, err := GetGkillServerAPI().GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("error at get account %s: %w", userID, err)
	}
	return account != nil, nil
}

func (gkillPluginCacheSource) PluginRepositories(_ context.Context, userID string) ([]reps.PluginRepository, error) {
	// GetPluginManager は走査（manifest.json の読み込み）だけで、プロセスは起動しない。
	return GetGkillServerAPI().GkillDAOManager.GetPluginManager(userID).GetPluginRepositories(), nil
}

// pluginBuildRunner はプラグイン1本を単独モードで起動して結果を返す。本番は runPluginBuildCache。
type pluginBuildRunner func(ctx context.Context, target pluginBuildTarget, userID string) (pluginBuildResult, error)

// GeneratePluginCache は userID のプラグイン（pluginName か all）を逐次に単独モードで起動してキャッシュを作る。
func GeneratePluginCache(ctx context.Context, userID string, pluginName string, out io.Writer) error {
	return generatePluginCache(ctx, userID, pluginName, gkillPluginCacheSource{}, runPluginBuildCache, out)
}

// generatePluginCache が本体。
//
// 利用者の実在確認はプラグインの走査より前に置く。走査（PluginManager.DiscoverPlugins）は
// $GKILL_HOME/plugins/{userID}/ を MkdirAll するので、打ち間違えた user_id でディレクトリが増える。
// プラグインは逐次に起動する。並列にすると stderr の進捗が混ざり、fitbit の並列パーサが
// 数 GB の ZIP を同時に開く。1本でも失敗したら残りを続けてから errors.Join で返す。
func generatePluginCache(ctx context.Context, userID string, pluginName string, src pluginCacheSource, run pluginBuildRunner, out io.Writer) error {
	targets, err := preparePluginBuildTargets(ctx, userID, pluginName, src)
	if err != nil {
		// 起動前の失敗（不正・未登録の user_id、走査失敗、無いプラグイン名）は何も起動していないので、
		// stdout の進捗行が1行も無い。main の log.Fatal はログファイルにしか書かず端末には exit 1 しか
		// 見えないため、stderr にも出す（add_tag のルール書式誤りと同じ扱い）。
		fmt.Fprintf(os.Stderr, "generate_plugin_cache: user id = %s: %v\n", userID, err)
		return err
	}
	if len(targets) == 0 {
		fmt.Fprintf(out, "no plugins: user id = %s\n", userID)
		return nil
	}

	var errs []error
	for _, target := range targets {
		fmt.Fprintf(out, "build plugin cache: user id = %s plugin = %s\n", userID, target.Name)
		started := time.Now()
		result, err := run(ctx, target, userID)
		elapsed := time.Since(started).Round(100 * time.Millisecond)
		if err != nil {
			fmt.Fprintf(out, "failed plugin cache: user id = %s plugin = %s elapsed = %s: %v\n", userID, target.Name, elapsed, err)
			errs = append(errs, fmt.Errorf("plugin %s: %w", target.Name, err))
			continue
		}
		fmt.Fprintf(out, "built plugin cache: user id = %s plugin = %s result = %s elapsed = %s\n", userID, target.Name, result, elapsed)
	}
	return errors.Join(errs...)
}

// preparePluginBuildTargets は起動前の検査をまとめて行い、起動する対象を返す。
// 順序は「user_id の形 → 利用者の実在 → プラグインの走査 → 名前の照合」で、
// 実在確認が走査より前なのは走査が plugins/{user}/ を MkdirAll するため。
func preparePluginBuildTargets(ctx context.Context, userID string, pluginName string, src pluginCacheSource) ([]pluginBuildTarget, error) {
	if !isSafeUserIDForPluginCache(userID) {
		return nil, fmt.Errorf("invalid user id: %q", userID)
	}
	exists, err := src.AccountExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("user id %s not found", userID)
	}

	plugins, err := src.PluginRepositories(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error at discover plugins user id = %s: %w", userID, err)
	}
	return selectPluginBuildTargets(plugins, pluginName)
}

// isSafeUserIDForPluginCache は user_id を子プロセスの引数とパス要素に使ってよいかを返す（ClearPluginCache と同じ規則）。
func isSafeUserIDForPluginCache(userID string) bool {
	if userID == "" || userID == "." || userID == ".." {
		return false
	}
	return !strings.ContainsAny(userID, `/\`)
}

// selectPluginBuildTargets は all なら発見順に全部、それ以外は manifest.Name の完全一致1本を返す。
// 見つからなければ利用可能な名前を列挙したエラー（何も起動しない）。
func selectPluginBuildTargets(plugins []reps.PluginRepository, pluginName string) ([]pluginBuildTarget, error) {
	targets := make([]pluginBuildTarget, 0, len(plugins))
	names := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		manifest := plugin.GetManifest()
		names = append(names, manifest.Name)
		if pluginName != generatePluginCacheAllPlugins && manifest.Name != pluginName {
			continue
		}
		pluginDir := plugin.GetPluginDir()
		targets = append(targets, pluginBuildTarget{
			Name:            manifest.Name,
			ExecPath:        reps.PluginExecutablePath(pluginDir, manifest.Executable),
			PluginDir:       pluginDir,
			ProtocolVersion: manifest.ProtocolVersion,
		})
	}
	if pluginName != generatePluginCacheAllPlugins && len(targets) == 0 {
		available := "(none)"
		if len(names) != 0 {
			available = strings.Join(names, ", ")
		}
		return nil, fmt.Errorf("plugin %q not found. available: %s", pluginName, available)
	}
	return targets, nil
}

// runPluginBuildCache はプラグイン1本を --gkill-build-cache 付きで起動し、終わるまで待つ。
//
// stdin は繋がない（nil = 空）。フラグを知らない古いバイナリが stdio ループに入っても
// EOF で自然に終わるので、ハングしない。stdout は結果行の判定に使うので捕捉し、
// stderr（プラグインの進捗・診断）はそのまま端末へ流す。タイムアウトは設けない
// （fitbit の初回構築は実測 155 秒で、環境により数倍ぶれる）。
func runPluginBuildCache(ctx context.Context, target pluginBuildTarget, userID string) (pluginBuildResult, error) {
	cmd := exec.CommandContext(ctx, target.ExecPath,
		"--gkill-plugin-dir", target.PluginDir,
		"--gkill-user-id", userID,
		"--gkill-protocol-version", target.ProtocolVersion,
		"--gkill-build-cache",
	)
	cmd.Stdin = nil
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return "", fmt.Errorf("error at start plugin %s (%s): %w", target.Name, target.ExecPath, err)
	}
	return classifyPluginBuildResult(stdout.String(), cmd.ProcessState.ExitCode())
}

// classifyPluginBuildResult は子プロセスの stdout と終了コードから結果を判定する。
//
// 判定は stdout 全体（前後の空白を除く）と結果行の完全一致。最終行だけ見る緩い判定にすると、
// stdout（プロトコルのチャネル）を汚したプラグインを見逃す。exit 0 で結果行が無いのは
// 「フラグを無視して stdio ループに入り stdin EOF で終わった」旧 SDK・独自実装のバイナリ、
// exit 2 は Go の flag が未知フラグで落ちた旧 SDK のバイナリ。どちらも静かに成功にしない。
func classifyPluginBuildResult(stdout string, exitCode int) (pluginBuildResult, error) {
	line := strings.TrimSpace(stdout)
	switch {
	case exitCode == 0 && line == string(pluginBuildResultBuilt):
		return pluginBuildResultBuilt, nil
	case exitCode == 0 && line == string(pluginBuildResultNoCache):
		return pluginBuildResultNoCache, nil
	case exitCode == 0 && line == "":
		return "", errors.New("no result line on stdout (the plugin ignored --gkill-build-cache: built with an old SDK, or not using the SDK)")
	case exitCode == 0:
		return "", fmt.Errorf("unexpected result line on stdout %q (want built or no_cache)", line)
	case exitCode == 2:
		return "", errors.New("exit 2: the plugin does not understand --gkill-build-cache (built with an old SDK; rebuild it)")
	default:
		return "", fmt.Errorf("build failed with exit %d (see the plugin's stderr above)", exitCode)
	}
}
