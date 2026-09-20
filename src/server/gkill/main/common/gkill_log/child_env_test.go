package gkill_log

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// TestParseLevelVocabulary は --log / GKILL_LOG_LEVEL の語彙がレベルへ写ること、
// 未知の値が error になることを固定する。InitNamed と子プロセスの両方がこの1本を使う。
func TestParseLevelVocabulary(t *testing.T) {
	cases := map[string]slog.Level{
		"trace_sql": TraceSQL,
		"trace":     Trace,
		"debug":     Debug,
		"access":    Access,
		"info":      Info,
		"warn":      Warn,
		"error":     Error,
		"none":      None,
		"ERROR":     Error,
		" info ":    Info,
	}
	for name, want := range cases {
		got, err := ParseLevel(name)
		if err != nil {
			t.Errorf("ParseLevel(%q): %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", name, got, want)
		}
	}
	if _, err := ParseLevel("bogus"); err == nil {
		t.Error("未知のレベル名で error が返らない")
	}
}

// TestInitNamedWithReturnsErrorInsteadOfPanic は InitNamedWith がディレクトリを作れないとき
// panic せず error を返し、slog の既定ロガーも package の router も差し替えないことを固定する。
//
// プラグインの子プロセスはこの経路でログを開く。ログ dir の都合でプラグインが死ぬと、
// 記録の読み取りとは無関係な理由で「rep は候補に出るのに0件」になる。
func TestInitNamedWithReturnsErrorInsteadOfPanic(t *testing.T) {
	tmpDir := t.TempDir()
	// ファイルの下にディレクトリは作れない
	blocker := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	originalRouter := router
	originalLogger := slog.Default()
	t.Cleanup(func() {
		router = originalRouter
		slog.SetDefault(originalLogger)
	})

	err := InitNamedWith(NamedOptions{
		LogDir: filepath.Join(blocker, "logs"),
		Prefix: "gkill_plugin_x",
		App:    "gkill_plugin",
		Level:  Info,
	})
	if err == nil {
		t.Fatal("作れないディレクトリで error が返らない")
	}
	if router != originalRouter {
		t.Error("失敗したのに package の router が差し替わっている")
	}
	if slog.Default() != originalLogger {
		t.Error("失敗したのに slog の既定ロガーが差し替わっている")
	}
}

// TestChildEnvRoundTrip は親が書き出した環境変数を子がそのまま読めることを固定する。
// 変数名を片方だけ変えると、子は黙って既定（error）へ倒れて --log debug が届かなくなる。
func TestChildEnvRoundTrip(t *testing.T) {
	t.Setenv(EnvLogLevel, "")
	t.Setenv(EnvLogRotateMaxBytes, "")
	t.Setenv(EnvLogRotateKeep, "")
	originalLevel := LogLevelFromCmd
	originalMaxBytes := gkill_options.LogRotateMaxBytes
	originalKeep := gkill_options.LogRotateKeep
	t.Cleanup(func() {
		LogLevelFromCmd = originalLevel
		gkill_options.LogRotateMaxBytes = originalMaxBytes
		gkill_options.LogRotateKeep = originalKeep
	})

	LogLevelFromCmd = "debug"
	gkill_options.LogRotateMaxBytes = 1234
	gkill_options.LogRotateKeep = 7
	if err := ExportEnvForChildProcesses(); err != nil {
		t.Fatal(err)
	}
	// 子プロセスではフラグを解析しないので gkill_options は既定値のまま。
	// そこから読んだ設定が親の値になることを見る。
	gkill_options.LogRotateMaxBytes = originalMaxBytes
	gkill_options.LogRotateKeep = originalKeep

	settings := ChildSettingsFromEnv()
	if settings.Level != Debug {
		t.Errorf("Level = %v, want Debug", settings.Level)
	}
	if settings.RotateMaxBytes != 1234 {
		t.Errorf("RotateMaxBytes = %d, want 1234", settings.RotateMaxBytes)
	}
	if settings.RotateKeep != 7 {
		t.Errorf("RotateKeep = %d, want 7", settings.RotateKeep)
	}
	if len(settings.Warnings) != 0 {
		t.Errorf("正しい値なのに Warnings がある: %v", settings.Warnings)
	}
}

// TestChildSettingsFromEnvDefaultsAndWarnings は環境変数が無いとき既定（error / 回転既定値）、
// 壊れているとき既定へ倒して Warnings で知らせる（止めない）ことを固定する。
func TestChildSettingsFromEnvDefaultsAndWarnings(t *testing.T) {
	t.Setenv(EnvLogLevel, "")
	t.Setenv(EnvLogRotateMaxBytes, "")
	t.Setenv(EnvLogRotateKeep, "")
	os.Unsetenv(EnvLogLevel)
	os.Unsetenv(EnvLogRotateMaxBytes)
	os.Unsetenv(EnvLogRotateKeep)

	settings := ChildSettingsFromEnv()
	if settings.Level != Error {
		t.Errorf("未設定の Level = %v, want Error", settings.Level)
	}
	if settings.RotateMaxBytes != gkill_options.LogRotateMaxBytes || settings.RotateKeep != gkill_options.LogRotateKeep {
		t.Errorf("未設定の回転 = (%d, %d), want (%d, %d)", settings.RotateMaxBytes, settings.RotateKeep, gkill_options.LogRotateMaxBytes, gkill_options.LogRotateKeep)
	}
	if len(settings.Warnings) != 0 {
		t.Errorf("未設定なのに Warnings がある: %v", settings.Warnings)
	}

	t.Setenv(EnvLogLevel, "bogus")
	t.Setenv(EnvLogRotateMaxBytes, "many")
	t.Setenv(EnvLogRotateKeep, "few")
	settings = ChildSettingsFromEnv()
	if settings.Level != Error {
		t.Errorf("壊れた Level = %v, want Error", settings.Level)
	}
	if len(settings.Warnings) != 3 {
		t.Errorf("Warnings = %d件 %v, want 3件", len(settings.Warnings), settings.Warnings)
	}
	for _, warning := range settings.Warnings {
		if !strings.Contains(warning, "GKILL_LOG_") {
			t.Errorf("Warnings に変数名が無い: %q", warning)
		}
	}
}

// TestInitExportsChildEnv は本体の Init() が子プロセス向けの環境変数を書き出すことを固定する。
// ここが抜けると、プラグインは常に既定の error で動き、--log debug で起動しても
// プラグイン側のファイルには何も増えない。
func TestInitExportsChildEnv(t *testing.T) {
	t.Setenv(EnvLogLevel, "")
	t.Setenv(EnvLogRotateMaxBytes, "")
	t.Setenv(EnvLogRotateKeep, "")
	tmpDir := t.TempDir()
	originalLogDir := gkill_options.LogDir
	originalLevel := LogLevelFromCmd
	originalRouter := router
	originalLogger := slog.Default()
	gkill_options.LogDir = tmpDir
	LogLevelFromCmd = "info"
	Init()
	initializedRouter := router
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
		_ = initializedRouter.Close()
		gkill_options.LogDir = originalLogDir
		LogLevelFromCmd = originalLevel
		router = originalRouter
	})

	if got := os.Getenv(EnvLogLevel); got != "info" {
		t.Errorf("%s = %q, want info", EnvLogLevel, got)
	}
	if got := os.Getenv(EnvLogRotateMaxBytes); got == "" {
		t.Errorf("%s が空", EnvLogRotateMaxBytes)
	}
	if got := os.Getenv(EnvLogRotateKeep); got == "" {
		t.Errorf("%s が空", EnvLogRotateKeep)
	}
}
