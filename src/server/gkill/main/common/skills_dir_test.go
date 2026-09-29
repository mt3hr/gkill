package common

// InitGkillOptions が SkillsDir を GkillHomeDir から派生させることの回帰テスト。
//
// gkill_options.SkillsDir の既定値は "$HOME/gkill/skills" の未展開文字列で、InitGkillOptions が
// "<GkillHomeDir>/skills" へ付け替えている（common.go）。その1行を消してもビルドも vet も通り、
// --gkill_home_dir を別の場所へ向けた起動（Android の APK は必ず渡す。MCP は GKILL_HOME を採る）で
// スキルだけがエラーも警告も出さずに $HOME/gkill/skills へ置かれる。
// gkill_options/option_test.go の TestDefaultSkillsDir は既定の文字列しか見ていないので、ここでは派生の側を固定する。
// 展開は NewGkillDAOManager が skills.NewStore へ渡すときと同じ式（filepath.Clean(os.ExpandEnv(...))）で見る。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// isolateGkillOptionsForSkillsDirTest は GkillHomeDir と派生オプションを退避し、終了時に元へ戻す。
// libc への TZ 適用（Android 専用）は差し替えて、環境変数 TZ を触らないようにする。
// 既定の $HOME も一時ディレクトリへ向け、そこへ落ちたときに他のテストの HOME と混ざらないようにする。
// 戻り値は隔離した環境での既定の置き場（$HOME/gkill/skills を展開したもの）。
func isolateGkillOptionsForSkillsDirTest(t *testing.T) string {
	t.Helper()
	t.Setenv("GKILL_HOME", os.Getenv("GKILL_HOME")) // InitGkillOptions が上書きするので終了時に戻す
	originalHomeDir := gkill_options.GkillHomeDir
	originalFn := applyLibcTimezoneFn
	t.Cleanup(func() {
		applyLibcTimezoneFn = originalFn
		gkill_options.GkillHomeDir = originalHomeDir
		InitGkillOptions() // 派生オプション（SkillsDir 等）を元の値へ戻す
	})
	applyLibcTimezoneFn = func(string) string { return "" }

	otherHome := t.TempDir()
	t.Setenv("HOME", otherHome)
	return filepath.Join(otherHome, "gkill", "skills")
}

// expandedSkillsDir は NewGkillDAOManager が skills.NewStore へ渡すのと同じ式で SkillsDir を展開する。
func expandedSkillsDir() string {
	return filepath.Clean(os.ExpandEnv(gkill_options.SkillsDir))
}

// assertSkillsDirUnderHome は SkillsDir が <home>/skills を指し、既定の $HOME/gkill/skills を指していないことを見る。
func assertSkillsDirUnderHome(t *testing.T, home string, defaultSkillsDir string) {
	t.Helper()
	got := expandedSkillsDir()
	if want := filepath.Join(home, "skills"); got != want {
		t.Fatalf("SkillsDir（展開後）= %q, want %q", got, want)
	}
	if strings.Contains(got, "$") || !filepath.IsAbs(got) {
		t.Fatalf("SkillsDir が展開済みの絶対パスでない: %q", got)
	}
	if got == defaultSkillsDir || strings.HasPrefix(got, defaultSkillsDir+string(filepath.Separator)) {
		t.Fatalf("SkillsDir が既定の $HOME/gkill/skills のまま: %q", got)
	}
	// 未展開の文字列も GkillHomeDir から派生している（他の派生ディレクトリと同じ組み立て）
	unexpandedWant := filepath.Clean(filepath.Join(gkill_options.GkillHomeDir, "skills"))
	if unexpandedGot := filepath.Clean(gkill_options.SkillsDir); unexpandedGot != unexpandedWant {
		t.Fatalf("SkillsDir（未展開）= %q, want %q（GkillHomeDir = %q）", unexpandedGot, unexpandedWant, gkill_options.GkillHomeDir)
	}
}

// newGkillHomeDirTestCmd は gkill_server / gkill の main.go と同じ束ね方
// （PersistentFlags の --gkill_home_dir → 実行時に InitGkillOptions）を持つコマンドを返す。
func newGkillHomeDirTestCmd(run func(cmd *cobra.Command)) *cobra.Command {
	root := &cobra.Command{
		Use: "root",
		Run: func(cmd *cobra.Command, _ []string) { run(cmd) },
	}
	root.PersistentFlags().StringVar(&gkill_options.GkillHomeDir, "gkill_home_dir", gkill_options.GkillHomeDir, "")
	return root
}

func TestInitGkillOptionsDerivesSkillsDirFromGkillHomeDir(t *testing.T) {
	t.Run("--gkill_home_dir で別の場所へ向けると skills もその下になる", func(t *testing.T) {
		defaultSkillsDir := isolateGkillOptionsForSkillsDirTest(t)
		home := t.TempDir()

		root := newGkillHomeDirTestCmd(func(*cobra.Command) { InitGkillOptions() })
		root.SetArgs([]string{"--gkill_home_dir", home})
		if err := root.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		assertSkillsDirUnderHome(t, home, defaultSkillsDir)
	})

	t.Run("環境変数を含むホームは展開した先の skills になる", func(t *testing.T) {
		defaultSkillsDir := isolateGkillOptionsForSkillsDirTest(t)
		home := t.TempDir()
		t.Setenv("GKILL_TEST_SKILLS_HOME", home)

		gkill_options.GkillHomeDir = "$GKILL_TEST_SKILLS_HOME/gkill"
		InitGkillOptions()

		assertSkillsDirUnderHome(t, filepath.Join(home, "gkill"), defaultSkillsDir)
		// プラグインへ継ぐ GKILL_HOME（InitGkillOptions が展開して書く）とも同じ場所を指す
		if got, want := expandedSkillsDir(), filepath.Join(os.Getenv("GKILL_HOME"), "skills"); got != want {
			t.Fatalf("SkillsDir = %q が GKILL_HOME 由来の %q と食い違う", got, want)
		}
	})

	t.Run("--gkill_home_dir 無しで GKILL_HOME を採る MCP の経路でも同じ", func(t *testing.T) {
		defaultSkillsDir := isolateGkillOptionsForSkillsDirTest(t)
		home := t.TempDir()
		t.Setenv("GKILL_HOME", home)

		root := newGkillHomeDirTestCmd(func(cmd *cobra.Command) {
			applyGkillHomeEnv(cmd) // mcp.go: フラグが明示されていなければ GKILL_HOME を採る
			InitGkillOptions()
		})
		root.SetArgs([]string{})
		if err := root.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		assertSkillsDirUnderHome(t, home, defaultSkillsDir)
	})
}
