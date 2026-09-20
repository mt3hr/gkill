package sdk

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// setupPluginLogTest は GKILL_HOME を一時ディレクトリに向け、stderr 側をバッファへ差し替え、
// 終了時にログを閉じて既定ロガーを戻す。戻り値は (gkill home, stderr バッファ)。
func setupPluginLogTest(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)
	t.Setenv(gkill_log.EnvLogLevel, "")
	t.Setenv(gkill_log.EnvLogRotateMaxBytes, "")
	t.Setenv(gkill_log.EnvLogRotateKeep, "")
	var stderr bytes.Buffer
	SetLogWriter(&stderr)
	originalLogger := slog.Default()
	t.Cleanup(func() {
		closeLogging()
		SetLogWriter(nil)
		slog.SetDefault(originalLogger)
	})
	return home, &stderr
}

func readLog(t *testing.T, home, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, "logs", name))
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", name, err)
	}
	return string(data)
}

// TestInitLoggingWritesUnderGkillHome は initLogging が $GKILL_HOME/logs/gkill_plugin_<name>*.log を開き、
// LogWarn が stderr とファイルの両方へ、LogInfo がファイルにだけ出ることを固定する。
//
// stderr 側の接頭辞行は gkill 本体の last_error リング（4KB）が読むので、Info で押し出さない。
func TestInitLoggingWritesUnderGkillHome(t *testing.T) {
	home, stderr := setupPluginLogTest(t)
	t.Setenv(gkill_log.EnvLogLevel, "info")
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_x")

	initLogging(pluginDir, "testuser")
	if fileLogger == nil {
		t.Fatalf("ファイルが開いていない。stderr: %s", stderr.String())
	}
	LogWarn("hello %d", 1)
	LogInfo("milestone %s", "done")
	LogDebug("hidden at info level")
	closeLogging()

	merged := readLog(t, home, "gkill_plugin_x.log")
	for _, want := range []string{`"app":"gkill_plugin"`, `"plugin":"gkill_plugin_x"`, `"user_id":"testuser"`, `"pid":`, `"msg":"hello 1"`, `"msg":"milestone done"`, `"msg":"plugin start"`} {
		if !strings.Contains(merged, want) {
			t.Errorf("統合ファイルに %s が無い: %s", want, merged)
		}
	}
	if strings.Contains(merged, "hidden at info level") {
		t.Errorf("info レベルなのに Debug が出ている: %s", merged)
	}
	// source はプラグイン側の呼び出し行（このテストファイル）を指す。log.go を指すと役に立たない。
	if !strings.Contains(merged, "plugin_log_test.go") {
		t.Errorf("source が呼び出し元を指していない: %s", merged)
	}
	if warn := readLog(t, home, "gkill_plugin_x_warn.log"); !strings.Contains(warn, `"msg":"hello 1"`) {
		t.Errorf("_warn.log に WARN 行が無い: %s", warn)
	}
	if info := readLog(t, home, "gkill_plugin_x_info.log"); !strings.Contains(info, `"msg":"milestone done"`) {
		t.Errorf("_info.log に Info 行が無い: %s", info)
	}

	if !strings.Contains(stderr.String(), "WARN: hello 1") {
		t.Errorf("stderr に接頭辞行が無い: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "milestone") {
		t.Errorf("Info が stderr に漏れている: %q", stderr.String())
	}
}

// TestInitLoggingPrefixDoesNotDouble は gkill_plugin_ で始まる名前を二重にしないこと、
// それ以外には付けることを固定する（gkill_plugin_uguisu*.log / gkill_plugin_gkill_example*.log）。
func TestInitLoggingPrefixDoesNotDouble(t *testing.T) {
	cases := map[string]string{
		"gkill_plugin_uguisu": "gkill_plugin_uguisu",
		"gkill_example":       "gkill_plugin_gkill_example",
	}
	for name, wantPrefix := range cases {
		if got := pluginLogPrefix(name); got != wantPrefix {
			t.Errorf("pluginLogPrefix(%q) = %q, want %q", name, got, wantPrefix)
		}
	}

	home, _ := setupPluginLogTest(t)
	initLogging(filepath.Join(home, "plugins", "testuser", "gkill_plugin_uguisu"), "testuser")
	closeLogging()
	if _, err := os.Stat(filepath.Join(home, "logs", "gkill_plugin_uguisu.log")); err != nil {
		t.Errorf("gkill_plugin_uguisu.log が無い: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "logs", "gkill_plugin_gkill_plugin_uguisu.log")); err == nil {
		t.Error("接頭辞が二重になったファイルが作られている")
	}
}

// TestInitLoggingWithoutHomeIsStderrOnly は GKILL_HOME が無く plugins/{user}/{name} の形でもないとき、
// ファイルを作らず stderr だけで続行する（panic も exit もしない）ことを固定する。手起動の型。
func TestInitLoggingWithoutHomeIsStderrOnly(t *testing.T) {
	_, stderr := setupPluginLogTest(t)
	t.Setenv("GKILL_HOME", "")
	os.Unsetenv("GKILL_HOME")
	loose := t.TempDir()

	initLogging(filepath.Join(loose, "somewhere", "gkill_plugin_x"), "testuser")
	if fileLogger != nil {
		t.Fatal("home が分からないのにファイルを開いている")
	}
	LogWarn("still on stderr")
	if !strings.Contains(stderr.String(), "WARN: still on stderr") {
		t.Errorf("stderr に出ていない: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "stderr only") {
		t.Errorf("ファイルを開いていない旨の警告が無い: %q", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(loose, "logs")); err == nil {
		t.Error("推定できないのに logs/ を作っている")
	}
}

// TestInitLoggingInfersHomeFromPluginDir は GKILL_HOME が無くても
// $GKILL_HOME/plugins/{user}/{name} の形から home を推定してファイルを開くことを固定する
// （PluginCacheDir と同じ解決。片方だけ推定するとキャッシュとログの置き場が食い違う）。
func TestInitLoggingInfersHomeFromPluginDir(t *testing.T) {
	home, _ := setupPluginLogTest(t)
	t.Setenv("GKILL_HOME", "")
	os.Unsetenv("GKILL_HOME")

	initLogging(filepath.Join(home, "plugins", "testuser", "gkill_plugin_x"), "testuser")
	closeLogging()
	if _, err := os.Stat(filepath.Join(home, "logs", "gkill_plugin_x.log")); err != nil {
		t.Errorf("推定した home にファイルが無い: %v", err)
	}
}

// TestInitLoggingBadLevelFallsBackToError は壊れた GKILL_LOG_LEVEL で止まらず、
// stderr に1行警告して error レベルで続行することを固定する。
func TestInitLoggingBadLevelFallsBackToError(t *testing.T) {
	home, stderr := setupPluginLogTest(t)
	t.Setenv(gkill_log.EnvLogLevel, "bogus")

	initLogging(filepath.Join(home, "plugins", "testuser", "gkill_plugin_x"), "testuser")
	if fileLogger == nil {
		t.Fatalf("壊れたレベルでファイルを開けていない。stderr: %s", stderr.String())
	}
	LogInfo("not at error level")
	LogError("kept")
	closeLogging()

	if !strings.Contains(stderr.String(), "GKILL_LOG_LEVEL") {
		t.Errorf("stderr に警告が無い: %q", stderr.String())
	}
	merged := readLog(t, home, "gkill_plugin_x.log")
	if strings.Contains(merged, "not at error level") {
		t.Errorf("error へ倒れていない: %s", merged)
	}
	if !strings.Contains(merged, `"msg":"kept"`) {
		t.Errorf("Error が出ていない: %s", merged)
	}
}

// TestRunLoopWritesAccessLineAndKeepsStdoutClean は access レベルで1コマンド1行が _access.log に残り、
// stdout には JSON の応答以外が1バイトも混ざらないことを固定する。
func TestRunLoopWritesAccessLineAndKeepsStdoutClean(t *testing.T) {
	home, _ := setupPluginLogTest(t)
	t.Setenv(gkill_log.EnvLogLevel, "access")
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_x")
	initLogging(pluginDir, "testuser")

	h := Handler{
		FindKyous: func(context.Context, Query, Config) ([]Kyou, error) {
			return []Kyou{{ID: "a"}, {ID: "b"}}, nil
		},
	}
	in := strings.NewReader(`{"id":"1","command":"find_kyous"}` + "\n" + `{"id":"2","command":"nope"}` + "\n" + `{"id":"3","command":"close"}` + "\n")
	var out bytes.Buffer
	if !runLoop(h, Config{}, pluginDir, "testuser", in, &out) {
		t.Fatal("close で true が返らない")
	}
	closeLogging()

	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
			t.Errorf("stdout に JSON 以外の行がある: %q", line)
		}
	}
	access := readLog(t, home, "gkill_plugin_x_access.log")
	if !strings.Contains(access, `"command":"find_kyous"`) || !strings.Contains(access, `"count":2`) {
		t.Errorf("_access.log に find_kyous の行が無い: %s", access)
	}
	if !strings.Contains(access, `"command":"nope"`) || !strings.Contains(access, `"error":"unknown command: nope"`) {
		t.Errorf("_access.log に失敗したコマンドの文言が無い: %s", access)
	}
	if info := readLog(t, home, "gkill_plugin_x_info.log"); !strings.Contains(info, `"reason":"close command"`) {
		t.Errorf("_info.log に plugin stop が無い: %s", info)
	}
}
