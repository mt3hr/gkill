package common

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md（MCP の不変条件の正本）
//
// gkill_server mcp --kind read|write|readwrite [--transport stdio|http] [--config <path>]
//
// MCP サーバは gkill_server のサブコマンド。実体は起動中の gkill_server への HTTP クライアントで
// （update_cache / add_tag と同じ型。gkill-cli-ops スキル）、DB を直接は読まない。
// stdio モードでは **stdout が JSON-RPC の専用チャネル** なので、ログは gkill_log の別名ファイル
// （logs/gkill_mcp_<kind>*.log）へだけ出し、stdout へは1バイトも書かない。
// 設定は $GKILL_HOME/configs/gkill_mcp.json（無ければ初回起動時に生成）。優先順位は
// フラグ > 環境変数 > ファイル > 既定値（mcp/config.go）。経緯: documents/adr/0631-mcp-lives-in-gkill-server.md

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
	"github.com/mt3hr/gkill/src/server/gkill/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpKind               string
	mcpTransport          string
	mcpConfigPath         string
	mcpSchemaBudgetUpdate bool
)

// MCPCmd は `gkill_server mcp`。
var MCPCmd = &cobra.Command{
	Use:           "mcp",
	Short:         "mcp --kind read|write|readwrite [--transport stdio|http] [--config <path>]",
	SilenceUsage:  true,
	SilenceErrors: true,
	// 親（ServerCmd）の PersistentPreRun は gkill_log.Init() で logs/gkill.log を開く。cobra は最寄りの
	// PersistentPreRun だけを走らせるので、ここで自前に初期化し、ログは設定を読んでから InitNamed で
	// 別名のファイル群を開く（stdout ミラーは常に無効）。
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		applyGkillHomeEnv(cmd)
		InitGkillOptions()
		threads.Init()
	},
	RunE: runMCP,
}

// MCPSchemaBudgetCmd は `gkill_server mcp schema-budget [--update]`（旧 npm run mcp:schema-budget）。
// tools/list のバイト量を計測し、予算ファイル（mcp/tool_schema_budget.json）と突き合わせる。
var MCPSchemaBudgetCmd = &cobra.Command{
	Use:           "schema-budget",
	Short:         "schema-budget [--update]",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		current := mcp.MeasureToolSchemaBytes()
		path := mcp.ToolSchemaBudgetPath()
		if mcpSchemaBudgetUpdate {
			if err := mcp.WriteToolSchemaBudget(current, path); err != nil {
				return err
			}
			for _, kind := range mcp.ServerKinds {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %d bytes\n", kind, current[kind])
			}
			fmt.Fprintf(cmd.OutOrStdout(), "updated %s\n", path)
			return nil
		}
		budget, err := mcp.ReadToolSchemaBudgetFile(path)
		if err != nil {
			return err
		}
		failed := false
		for _, row := range mcp.CompareToolSchemaBudget(current, budget) {
			fmt.Fprintln(cmd.OutOrStdout(), mcp.DescribeBudgetRow(row))
			if row.Verdict == "over" {
				failed = true
			}
		}
		if failed {
			return errors.New("tools/list exceeds the recorded budget; trim descriptions or run `gkill_server mcp schema-budget --update`")
		}
		return nil
	},
}

func init() {
	MCPCmd.Flags().StringVar(&mcpKind, "kind", "", "read | write | readwrite (required)")
	MCPCmd.Flags().StringVar(&mcpTransport, "transport", "", "stdio | http (default: MCP_TRANSPORT, then the config file, then stdio)")
	MCPCmd.Flags().StringVar(&mcpConfigPath, "config", "", "path of the MCP config file (default: $GKILL_HOME/configs/gkill_mcp.json)")
	MCPSchemaBudgetCmd.Flags().BoolVar(&mcpSchemaBudgetUpdate, "update", false, "rewrite tool_schema_budget.json with the current sizes")
	MCPCmd.AddCommand(MCPSchemaBudgetCmd)
}

// applyGkillHomeEnv は --gkill_home_dir が明示されていないとき GKILL_HOME を採る
// （旧実装は GKILL_HOME || $HOME/gkill だった。InitGkillOptions はフラグ値で GKILL_HOME を上書きするので、その前に行う）。
func applyGkillHomeEnv(cmd *cobra.Command) {
	if cmd.Flags().Changed("gkill_home_dir") {
		return
	}
	if home := os.Getenv("GKILL_HOME"); home != "" {
		gkill_options.GkillHomeDir = home
	}
}

// resolveMCPSettings は設定ファイルと環境変数とフラグから起動設定を決める（RunE から分けてテストできるようにする）。
func resolveMCPSettings(cmd *cobra.Command, kind, transportFlag, configPath string) (mcp.StartSpec, mcp.Settings, string, bool, error) {
	if kind == "" {
		return mcp.StartSpec{}, mcp.Settings{}, "", false, errors.New("--kind is required (read | write | readwrite)")
	}
	spec := mcp.StartSpecFor(kind)
	if spec.Kind == "" {
		return mcp.StartSpec{}, mcp.Settings{}, "", false, fmt.Errorf("--kind must be one of read, write, readwrite (got %q)", kind)
	}
	if configPath == "" {
		configPath = mcp.DefaultConfigPath(os.ExpandEnv(gkill_options.ConfigDir))
	}
	cfg, created, err := mcp.LoadOrCreateConfig(configPath)
	if err != nil {
		return mcp.StartSpec{}, mcp.Settings{}, configPath, false, err
	}
	flags := mcp.FlagOverrides{Transport: transportFlag}
	// 親の --log は「明示されたときだけ」最優先（既定値 error を MCP の既定 access より優先しない）。
	if cmd != nil && cmd.Flags().Changed("log") {
		flags.LogLevel = gkill_log.LogLevelFromCmd
	}
	settings, err := mcp.ResolveSettings(cfg, kind, flags, os.Getenv)
	if err != nil {
		return mcp.StartSpec{}, mcp.Settings{}, configPath, created, err
	}
	return spec, settings, configPath, created, nil
}

func runMCP(cmd *cobra.Command, _ []string) error {
	spec, settings, configPath, created, err := resolveMCPSettings(cmd, mcpKind, mcpTransport, mcpConfigPath)
	if err != nil {
		return err
	}

	// ログは設定が決まってから開く。ファイル名は種別ごと（logs/gkill_mcp_<kind>*.log）。
	gkill_log.LogLevelFromCmd = settings.LogLevelName
	gkill_log.InitNamed(spec.LogPrefix, "gkill_mcp", "kind", spec.Kind)
	gkill_log.SetStdoutMirror(false)
	logger := mcp.NewLogger(slog.Default())
	if created {
		logger.Info("config_created", "path", configPath)
	}

	// サーバが名乗る版は本体と同じ出所（embed の version.json）。
	if version, err := api.GetVersion(); err == nil && version != nil && version.Version != "" {
		mcp.ServerVersion = version.Version
	} else {
		logger.Warn("version_unavailable", "error", fmt.Sprintf("%v", err))
	}

	mcp.ApplySettings(settings)
	client := mcp.NewGkillClient(settings.Client)

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer func() {
		_ = gkill_log.Close()
	}()

	return mcp.Start(ctx, spec, mcp.StartOptions{
		Transport:      settings.Transport,
		Client:         client,
		Log:            logger,
		LogLevelName:   settings.LogLevelName,
		Port:           settings.Port,
		BindAddr:       settings.BindAddr,
		OAuthIssuer:    settings.OAuthIssuer,
		OAuthStatePath: mcpOAuthStatePath(spec),
	})
}

// mcpOAuthStatePath は OAuth の状態ファイル（DCR クライアント・refresh token）の置き場所。
// $GKILL_HOME/configs/ 配下、名前は種別ごと（spec.OAuthStateFileName）。ConfigDir は未展開の
// "$HOME/gkill/configs" なので、ここで環境変数を展開する。
func mcpOAuthStatePath(spec mcp.StartSpec) string {
	return filepath.Join(os.ExpandEnv(gkill_options.ConfigDir), spec.OAuthStateFileName)
}
