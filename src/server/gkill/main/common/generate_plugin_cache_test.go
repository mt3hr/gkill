package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// generate_plugin_cache はプラグインバイナリを --gkill-build-cache 付きで起動して待つ。
// 外部バイナリをビルドせずにこれを検証するため、dao/reps のプラグインテストと同じく
// 「テストバイナリ自身を再execする」方式をとる。TestMain が環境変数を見て偽プラグインとして振る舞う。

const envFakeBuildPlugin = "GKILL_TEST_FAKE_BUILD_PLUGIN"

const (
	fakeBuildBuilt      = "built"       // 結果行 built を出して exit 0
	fakeBuildNoCache    = "no_cache"    // 結果行 no_cache を出して exit 0
	fakeBuildIgnoreFlag = "ignore_flag" // フラグを無視して stdin を EOF まで読み、何も出さず exit 0（旧 SDK・独自実装）
	fakeBuildExit2      = "exit2"       // Go の flag が未知フラグで落ちたときの形（stderr に usage、exit 2）
	fakeBuildFail       = "fail"        // 構築失敗（stderr にエラー、exit 1）
	fakeBuildDirty      = "dirty"       // 結果行の前に余計な行を stdout へ出す
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(envFakeBuildPlugin); mode != "" {
		runFakeBuildPlugin(mode)
		return
	}
	os.Exit(m.Run())
}

// runFakeBuildPlugin は偽プラグイン。--gkill-build-cache を受け取っていなければ、
// どのモードでも旧挙動（stdin を EOF まで読んで exit 0・stdout 空）になる。
// built がモードどおりに出るのは CLI がフラグを渡しているときだけなので、
// 「CLI が --gkill-build-cache を渡していること」も同時に固定される。
func runFakeBuildPlugin(mode string) {
	if !slices.Contains(os.Args[1:], "--gkill-build-cache") {
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	switch mode {
	case fakeBuildBuilt:
		fmt.Println("built")
	case fakeBuildNoCache:
		fmt.Println("no_cache")
	case fakeBuildIgnoreFlag:
		_, _ = io.Copy(io.Discard, os.Stdin)
	case fakeBuildExit2:
		fmt.Fprintln(os.Stderr, "flag provided but not defined: -gkill-build-cache")
		os.Exit(2)
	case fakeBuildFail:
		fmt.Fprintln(os.Stderr, "ERROR: build cache: boom")
		os.Exit(1)
	case fakeBuildDirty:
		fmt.Println("scanning 3 files")
		fmt.Println("built")
	}
	os.Exit(0)
}

// TestGeneratePluginCacheCmdRequiresPluginAndUser は引数が2つ未満なら usage を出して終わり、
// サーバ側の初期化（InitGkillServerAPI = 利用者の configs DB を開く）へ進まないことを固定する。
// 引数の並びは clear_cache <mode> <user_id...> と同じ「<plugin_name|all> <user_id...>」。
func TestGeneratePluginCacheCmdRequiresPluginAndUser(t *testing.T) {
	if GeneratePluginCacheCmd.Use != "generate_plugin_cache" {
		t.Fatalf("Use = %q, want generate_plugin_cache", GeneratePluginCacheCmd.Use)
	}
	for _, args := range [][]string{{}, {"all"}} {
		var out bytes.Buffer
		GeneratePluginCacheCmd.SetOut(&out)
		GeneratePluginCacheCmd.SetErr(&out)
		t.Cleanup(func() {
			GeneratePluginCacheCmd.SetOut(nil)
			GeneratePluginCacheCmd.SetErr(nil)
		})
		before := gkillServerAPI
		err := GeneratePluginCacheCmd.RunE(GeneratePluginCacheCmd, args)
		if err != nil {
			t.Errorf("args=%v: usage を出すだけのはずがエラー: %v", args, err)
		}
		if !strings.Contains(out.String(), "generate_plugin_cache") {
			t.Errorf("args=%v: usage が出ていない: %q", args, out.String())
		}
		if gkillServerAPI != before {
			t.Errorf("args=%v: 引数不足なのに InitGkillServerAPI まで進んだ", args)
		}
	}
}

// 結果の分類。stdout 全体の完全一致で判定し、exit 0 で結果行が無いもの・余計な出力があるものを
// 成功にしないことを固定する（静かに成功に見える旧バイナリを防ぐ唯一の判定）。
func TestClassifyPluginBuildResult(t *testing.T) {
	cases := []struct {
		name     string
		stdout   string
		exitCode int
		want     pluginBuildResult
		errWord  string
	}{
		{name: "built", stdout: "built\n", exitCode: 0, want: pluginBuildResultBuilt},
		{name: "no_cache", stdout: "no_cache\n", exitCode: 0, want: pluginBuildResultNoCache},
		{name: "前後の空白は許す", stdout: "  built \r\n", exitCode: 0, want: pluginBuildResultBuilt},
		{name: "exit 0 で空 = フラグを無視した旧バイナリ", stdout: "", exitCode: 0, errWord: "no result line"},
		{name: "exit 0 で余計な出力", stdout: "scanning\nbuilt\n", exitCode: 0, errWord: "unexpected result line"},
		{name: "exit 2 = 未知フラグ", stdout: "", exitCode: 2, errWord: "exit 2"},
		{name: "exit 1 = 構築失敗", stdout: "", exitCode: 1, errWord: "exit 1"},
		{name: "exit 1 でも built を出していたら失敗", stdout: "built\n", exitCode: 1, errWord: "exit 1"},
		{name: "その他の終了コード", stdout: "", exitCode: 3, errWord: "exit 3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := classifyPluginBuildResult(c.stdout, c.exitCode)
			if c.errWord == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != c.want {
					t.Errorf("result = %q, want %q", got, c.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("エラーになるべき（result = %q）", got)
			}
			if !strings.Contains(err.Error(), c.errWord) {
				t.Errorf("error = %q, want containing %q", err.Error(), c.errWord)
			}
		})
	}
}

// 偽プラグインを実際に起動して、フラグの受け渡し・stdout の捕捉・stdin nil（ハングしない）・
// 終了コードの分類が繋がっていることを確認する。
func TestRunPluginBuildCache_HelperProcess(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	target := pluginBuildTarget{
		Name:            "fake_plugin",
		ExecPath:        exe,
		PluginDir:       t.TempDir(),
		ProtocolVersion: "1",
	}
	cases := []struct {
		mode    string
		want    pluginBuildResult
		errWord string
	}{
		{mode: fakeBuildBuilt, want: pluginBuildResultBuilt},
		{mode: fakeBuildNoCache, want: pluginBuildResultNoCache},
		{mode: fakeBuildIgnoreFlag, errWord: "no result line"},
		{mode: fakeBuildExit2, errWord: "exit 2"},
		{mode: fakeBuildFail, errWord: "exit 1"},
		{mode: fakeBuildDirty, errWord: "unexpected result line"},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			t.Setenv(envFakeBuildPlugin, c.mode)
			got, err := runPluginBuildCache(context.Background(), target, "testuser")
			if c.errWord == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != c.want {
					t.Errorf("result = %q, want %q", got, c.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("エラーになるべき（result = %q）", got)
			}
			if !strings.Contains(err.Error(), c.errWord) {
				t.Errorf("error = %q, want containing %q", err.Error(), c.errWord)
			}
		})
	}
}

func TestRunPluginBuildCache_MissingExecutableIsError(t *testing.T) {
	target := pluginBuildTarget{
		Name:            "missing",
		ExecPath:        filepathJoinNoExist(t),
		PluginDir:       t.TempDir(),
		ProtocolVersion: "1",
	}
	if _, err := runPluginBuildCache(context.Background(), target, "testuser"); err == nil {
		t.Fatal("実行ファイルが無いのにエラーにならない")
	}
}

func filepathJoinNoExist(t *testing.T) string {
	t.Helper()
	return t.TempDir() + string(os.PathSeparator) + "no_such_plugin_binary"
}

// ── 対象の選択 ──

// stubPluginRepository は PluginRepository のうち generate_plugin_cache が使う2つだけを持つ。
type stubPluginRepository struct {
	reps.PluginRepository
	manifest gkill_plugin.PluginManifest
	dir      string
}

func (s stubPluginRepository) GetManifest() gkill_plugin.PluginManifest { return s.manifest }
func (s stubPluginRepository) GetPluginDir() string                     { return s.dir }

func stubPlugins() []reps.PluginRepository {
	mk := func(name string) reps.PluginRepository {
		return stubPluginRepository{
			manifest: gkill_plugin.PluginManifest{Name: name, Executable: name, ProtocolVersion: "1"},
			dir:      "/plugins/testuser/" + name,
		}
	}
	return []reps.PluginRepository{mk("gkill_plugin_a"), mk("gkill_plugin_b"), mk("gkill_plugin_c")}
}

func TestSelectPluginBuildTargets(t *testing.T) {
	plugins := stubPlugins()

	t.Run("all は発見順に全部", func(t *testing.T) {
		targets, err := selectPluginBuildTargets(plugins, "all")
		if err != nil {
			t.Fatal(err)
		}
		names := []string{}
		for _, target := range targets {
			names = append(names, target.Name)
		}
		if !slices.Equal(names, []string{"gkill_plugin_a", "gkill_plugin_b", "gkill_plugin_c"}) {
			t.Errorf("targets = %v", names)
		}
		want := reps.PluginExecutablePath("/plugins/testuser/gkill_plugin_a", "gkill_plugin_a")
		if targets[0].ExecPath != want {
			t.Errorf("ExecPath = %q, want %q（reps.PluginExecutablePath を通す）", targets[0].ExecPath, want)
		}
		if targets[0].PluginDir != "/plugins/testuser/gkill_plugin_a" || targets[0].ProtocolVersion != "1" {
			t.Errorf("PluginDir / ProtocolVersion が manifest から来ていない: %+v", targets[0])
		}
	})

	t.Run("名前指定は完全一致1本", func(t *testing.T) {
		targets, err := selectPluginBuildTargets(plugins, "gkill_plugin_b")
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 1 || targets[0].Name != "gkill_plugin_b" {
			t.Errorf("targets = %+v", targets)
		}
	})

	t.Run("無い名前は利用可能名を列挙してエラー", func(t *testing.T) {
		_, err := selectPluginBuildTargets(plugins, "gkill_plugin_x")
		if err == nil {
			t.Fatal("エラーになるべき")
		}
		for _, name := range []string{"gkill_plugin_a", "gkill_plugin_b", "gkill_plugin_c"} {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("error = %q, want containing %q", err.Error(), name)
			}
		}
	})

	t.Run("all でプラグイン0本は空で成功", func(t *testing.T) {
		targets, err := selectPluginBuildTargets(nil, "all")
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 0 {
			t.Errorf("targets = %+v", targets)
		}
	})
}

// ── 本体の流れ ──

type fakePluginCacheSource struct {
	exists    bool
	existsErr error
	plugins   []reps.PluginRepository
	listCalls int
}

func (f *fakePluginCacheSource) AccountExists(context.Context, string) (bool, error) {
	return f.exists, f.existsErr
}

func (f *fakePluginCacheSource) PluginRepositories(context.Context, string) ([]reps.PluginRepository, error) {
	f.listCalls++
	return f.plugins, nil
}

// 存在しない user_id はプラグインの走査より前に弾く。走査は plugins/<user>/ を MkdirAll するので、
// 打ち間違えのたびにディレクトリが増える。
func TestGeneratePluginCache_RejectsUnknownUserBeforeDiscovery(t *testing.T) {
	src := &fakePluginCacheSource{exists: false, plugins: stubPlugins()}
	runs := 0
	run := func(context.Context, pluginBuildTarget, string) (pluginBuildResult, error) {
		runs++
		return pluginBuildResultBuilt, nil
	}
	var out bytes.Buffer
	err := generatePluginCache(context.Background(), "nosuchuser", "all", src, run, &out)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v, want user not found", err)
	}
	if src.listCalls != 0 {
		t.Errorf("利用者が居ないのにプラグインを走査した（plugins/<typo>/ が作られる）: %d", src.listCalls)
	}
	if runs != 0 {
		t.Errorf("起動してはいけない: %d", runs)
	}
}

func TestGeneratePluginCache_RejectsUnsafeUserID(t *testing.T) {
	for _, userID := range []string{"", ".", "..", "a/b", `a\b`} {
		src := &fakePluginCacheSource{exists: true, plugins: stubPlugins()}
		run := func(context.Context, pluginBuildTarget, string) (pluginBuildResult, error) {
			t.Errorf("%q: 起動してはいけない", userID)
			return "", nil
		}
		var out bytes.Buffer
		if err := generatePluginCache(context.Background(), userID, "all", src, run, &out); err == nil {
			t.Errorf("%q: エラーになるべき", userID)
		}
		if src.listCalls != 0 {
			t.Errorf("%q: 走査してはいけない", userID)
		}
	}
}

// 1本が失敗しても残りは続け、最後にまとめて返す（generate_thumb_cache 等と同じ約束）。
func TestGeneratePluginCache_ContinuesAfterFailureAndJoinsErrors(t *testing.T) {
	src := &fakePluginCacheSource{exists: true, plugins: stubPlugins()}
	started := []string{}
	run := func(_ context.Context, target pluginBuildTarget, userID string) (pluginBuildResult, error) {
		started = append(started, target.Name)
		if userID != "testuser" {
			t.Errorf("user id = %q, want testuser", userID)
		}
		switch target.Name {
		case "gkill_plugin_a":
			return "", errors.New("boom")
		case "gkill_plugin_b":
			return pluginBuildResultNoCache, nil
		default:
			return pluginBuildResultBuilt, nil
		}
	}
	var out bytes.Buffer
	err := generatePluginCache(context.Background(), "testuser", "all", src, run, &out)
	if err == nil || !strings.Contains(err.Error(), "gkill_plugin_a") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v, want gkill_plugin_a の失敗を含む", err)
	}
	if !slices.Equal(started, []string{"gkill_plugin_a", "gkill_plugin_b", "gkill_plugin_c"}) {
		t.Errorf("失敗の後も残りを続けるべき: %v", started)
	}
	text := out.String()
	for _, want := range []string{
		"build plugin cache: user id = testuser plugin = gkill_plugin_a",
		"failed plugin cache: user id = testuser plugin = gkill_plugin_a",
		"built plugin cache: user id = testuser plugin = gkill_plugin_b result = no_cache",
		"built plugin cache: user id = testuser plugin = gkill_plugin_c result = built",
		"elapsed = ",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("stdout に %q が無い:\n%s", want, text)
		}
	}
}

func TestGeneratePluginCache_UnknownPluginNameRunsNothing(t *testing.T) {
	src := &fakePluginCacheSource{exists: true, plugins: stubPlugins()}
	runs := 0
	run := func(context.Context, pluginBuildTarget, string) (pluginBuildResult, error) {
		runs++
		return pluginBuildResultBuilt, nil
	}
	var out bytes.Buffer
	err := generatePluginCache(context.Background(), "testuser", "gkill_plugin_x", src, run, &out)
	if err == nil {
		t.Fatal("エラーになるべき")
	}
	if runs != 0 {
		t.Errorf("無い名前で何かを起動した: %d", runs)
	}
}

func TestGeneratePluginCache_NoPluginsIsSuccess(t *testing.T) {
	src := &fakePluginCacheSource{exists: true, plugins: nil}
	run := func(context.Context, pluginBuildTarget, string) (pluginBuildResult, error) {
		t.Error("起動してはいけない")
		return "", nil
	}
	var out bytes.Buffer
	if err := generatePluginCache(context.Background(), "testuser", "all", src, run, &out); err != nil {
		t.Fatalf("プラグイン0本は成功扱い: %v", err)
	}
	if !strings.Contains(out.String(), "no plugins: user id = testuser") {
		t.Errorf("stdout = %q", out.String())
	}
}
