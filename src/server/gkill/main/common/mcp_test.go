package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/mt3hr/gkill/src/server/gkill/mcp"
	"github.com/spf13/cobra"
)

// gkill_server mcp の配線（フラグの解決・設定ファイルの生成・stdout を汚さないこと）。
// サーバそのものの振る舞いは gkill/mcp 側のテストが持つ。

func TestMCPCmdIsWired(t *testing.T) {
	if MCPCmd.Use != "mcp" {
		t.Fatalf("Use = %q", MCPCmd.Use)
	}
	for _, name := range []string{"kind", "transport", "config"} {
		if MCPCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s is missing", name)
		}
	}
	found := false
	for _, sub := range MCPCmd.Commands() {
		if sub.Use == "schema-budget" {
			found = true
			if sub.Flags().Lookup("update") == nil {
				t.Errorf("schema-budget lacks --update")
			}
		}
	}
	if !found {
		t.Errorf("schema-budget subcommand is missing")
	}
	if !MCPCmd.SilenceUsage || !MCPCmd.SilenceErrors {
		t.Errorf("usage / errors must go to the caller (main logs them), not to stdout")
	}
}

// newMCPTestCmd は --log を持つ親にぶら下げた mcp コマンド（Changed の判定を本物と同じ経路で行う）。
func newMCPTestCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().StringVar(&gkill_log.LogLevelFromCmd, "log", gkill_log.LogLevelFromCmd, "")
	root.PersistentFlags().StringVar(&gkill_options.GkillHomeDir, "gkill_home_dir", gkill_options.GkillHomeDir, "")
	child := &cobra.Command{Use: "mcp", RunE: func(*cobra.Command, []string) error { return nil }}
	root.AddCommand(child)
	root.SetArgs(append([]string{"mcp"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return child
}

func TestResolveMCPSettings(t *testing.T) {
	originalLevel := gkill_log.LogLevelFromCmd
	originalHome := gkill_options.GkillHomeDir
	originalConfigDir := gkill_options.ConfigDir
	t.Cleanup(func() {
		gkill_log.LogLevelFromCmd = originalLevel
		gkill_options.GkillHomeDir = originalHome
		gkill_options.ConfigDir = originalConfigDir
	})
	for _, key := range []string{"MCP_TRANSPORT", "MCP_LOG", "MCP_PORT", "GKILL_BASE_URL", "GKILL_HOME"} {
		t.Setenv(key, "")
	}

	t.Run("--kind is required and validated", func(t *testing.T) {
		cmd := newMCPTestCmd(t)
		if _, _, _, _, err := resolveMCPSettings(cmd, "", "", ""); err == nil || !strings.Contains(err.Error(), "--kind is required") {
			t.Fatalf("err = %v", err)
		}
		if _, _, _, _, err := resolveMCPSettings(cmd, "admin", "", ""); err == nil || !strings.Contains(err.Error(), "--kind must be one of") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("creates the config under the config dir on first use and reports the path", func(t *testing.T) {
		dir := t.TempDir()
		gkill_options.ConfigDir = dir
		cmd := newMCPTestCmd(t)
		spec, settings, path, created, err := resolveMCPSettings(cmd, "read", "", "")
		if err != nil {
			t.Fatal(err)
		}
		if !created {
			t.Fatalf("config was not created")
		}
		if path != filepath.Join(dir, mcp.ConfigFileName) {
			t.Fatalf("path = %s", path)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("config file missing: %v", err)
		}
		if spec.Kind != "read" || settings.Port != 8808 || settings.Transport != "stdio" || settings.LogLevelName != "access" {
			t.Fatalf("unexpected settings: %+v / %+v", spec, settings)
		}
	})

	t.Run("the parent --log wins only when it was given explicitly", func(t *testing.T) {
		dir := t.TempDir()
		gkill_options.ConfigDir = dir
		t.Setenv("MCP_LOG", "debug")
		// 明示なし: 既定値 error は無視され、MCP_LOG が効く
		cmd := newMCPTestCmd(t)
		_, settings, _, _, err := resolveMCPSettings(cmd, "read", "", "")
		if err != nil {
			t.Fatal(err)
		}
		if settings.LogLevelName != "debug" {
			t.Fatalf("log level = %s, want debug (MCP_LOG)", settings.LogLevelName)
		}
		// 明示あり: --log warn がすべてに優先する
		cmd = newMCPTestCmd(t, "--log", "warn")
		_, settings, _, _, err = resolveMCPSettings(cmd, "read", "", "")
		if err != nil {
			t.Fatal(err)
		}
		if settings.LogLevelName != "warn" {
			t.Fatalf("log level = %s, want warn (--log)", settings.LogLevelName)
		}
	})

	t.Run("--transport and --config are honored", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "custom.json")
		cmd := newMCPTestCmd(t)
		_, settings, path, _, err := resolveMCPSettings(cmd, "readwrite", "http", configPath)
		if err != nil {
			t.Fatal(err)
		}
		if path != configPath || settings.Transport != "http" || settings.Port != 8810 {
			t.Fatalf("path=%s transport=%s port=%d", path, settings.Transport, settings.Port)
		}
	})
}

func TestApplyGkillHomeEnv(t *testing.T) {
	originalHome := gkill_options.GkillHomeDir
	t.Cleanup(func() { gkill_options.GkillHomeDir = originalHome })

	t.Run("GKILL_HOME is used when --gkill_home_dir is not given", func(t *testing.T) {
		gkill_options.GkillHomeDir = "$HOME/gkill"
		t.Setenv("GKILL_HOME", filepath.Join("C:", "tmp", "gkill_home_for_test"))
		cmd := newMCPTestCmd(t)
		applyGkillHomeEnv(cmd)
		if gkill_options.GkillHomeDir != os.Getenv("GKILL_HOME") {
			t.Fatalf("GkillHomeDir = %s", gkill_options.GkillHomeDir)
		}
	})

	t.Run("an explicit --gkill_home_dir beats GKILL_HOME", func(t *testing.T) {
		t.Setenv("GKILL_HOME", filepath.Join("C:", "tmp", "gkill_home_for_test"))
		cmd := newMCPTestCmd(t, "--gkill_home_dir", filepath.Join("C:", "tmp", "explicit"))
		applyGkillHomeEnv(cmd)
		if gkill_options.GkillHomeDir != filepath.Join("C:", "tmp", "explicit") {
			t.Fatalf("GkillHomeDir = %s", gkill_options.GkillHomeDir)
		}
	})
}

// OAuth の状態ファイルは $GKILL_HOME/configs/ 配下に種別ごとの名前で置く（旧 Node 実装と同じ場所・同じ名前。
// 引き継ぎのため）。ConfigDir の環境変数は展開してから使う。
func TestMCPOAuthStatePathIsUnderExpandedConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GKILL_TEST_MCP_HOME", home)
	original := gkill_options.ConfigDir
	gkill_options.ConfigDir = "$GKILL_TEST_MCP_HOME/gkill/configs"
	t.Cleanup(func() { gkill_options.ConfigDir = original })

	for _, kind := range []string{"read", "write", "readwrite"} {
		spec := mcp.StartSpecFor(kind)
		got := mcpOAuthStatePath(spec)
		want := filepath.Join(home, "gkill", "configs", "mcp_oauth_"+kind+"_state.json")
		if got != want {
			t.Errorf("kind=%s: path = %q, want %q", kind, got, want)
		}
		if strings.Contains(got, "$") {
			t.Errorf("kind=%s: 環境変数が未展開: %q", kind, got)
		}
	}
}
