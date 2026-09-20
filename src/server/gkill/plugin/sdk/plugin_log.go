package sdk

// 編集前に読む: .claude/skills/gkill-plugin/SKILL.md（プラグインのログの置き場と stdout 禁止の正本）

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// fileLogger は initLogging が開いた gkill_log のロガー。nil ならファイルは開いていない
// （手起動で GKILL_HOME が無い、ログ dir が作れない、Run に入る前）。
// nil のあいだ LogWarn / LogError は stderr にだけ出て、LogInfo / LogDebug は捨てられる。
var fileLogger *slog.Logger

const (
	// logAppName は静的フィールド app の値。本体は gkill、MCP は gkill_mcp。
	logAppName = "gkill_plugin"
	// logPrefixBase はファイル名の接頭辞。<prefix>.log と <prefix>_<level>.log が
	// $GKILL_HOME/logs に並ぶ（本体の gkill*.log、MCP の gkill_mcp_<kind>*.log と同じ置き場）。
	logPrefixBase = "gkill_plugin_"
)

// pluginLogPrefix はプラグイン名からファイル名の接頭辞を作る。
// 同梱プラグインは名前が既に gkill_plugin_ で始まるので二重にしない
// （gkill_plugin_fitbit → gkill_plugin_fitbit*.log、gkill_example → gkill_plugin_gkill_example*.log）。
func pluginLogPrefix(pluginName string) string {
	return logPrefixBase + strings.TrimPrefix(pluginName, logPrefixBase)
}

// pluginNameOf は --gkill-plugin-dir（$GKILL_HOME/plugins/{userID}/{pluginName}）の末尾＝
// manifest の name を返す。空や "." のような値は "" にする。
func pluginNameOf(pluginDir string) string {
	if pluginDir == "" {
		return ""
	}
	name := filepath.Base(filepath.Clean(pluginDir))
	if !IsSafePathElement(name) {
		return ""
	}
	return name
}

// initLogging は $GKILL_HOME/logs/gkill_plugin_<name>*.log を開く。Run が flag の解析直後に呼ぶ。
//
// レベルと回転は gkill 本体が --log / --log_rotate_* から環境変数（gkill_log.EnvLog*）へ書き出した値。
// 壊れていれば既定（error）へ倒して stderr に1行残す。GKILL_HOME が無く plugins/{user}/{name} の
// 形からも home を推定できないとき、またはログ dir を作れないときは、**stderr だけで続行する**
// （ログの都合でプラグインを止めると「rep は候補に出るのに0件」になる）。
// panic も os.Exit もしない。
func initLogging(pluginDir, userID string) {
	name := pluginNameOf(pluginDir)
	home := gkillHomeDir(pluginDir)
	if name == "" || home == "" {
		LogWarn("plugin log files are not opened (gkill home is unknown; set GKILL_HOME or pass --gkill-plugin-dir under $GKILL_HOME/plugins). continuing with stderr only")
		return
	}
	settings := gkill_log.ChildSettingsFromEnv()
	for _, warning := range settings.Warnings {
		LogWarn("%s", warning)
	}
	err := gkill_log.InitNamedWith(gkill_log.NamedOptions{
		LogDir:         filepath.Join(home, "logs"),
		Prefix:         pluginLogPrefix(name),
		App:            logAppName,
		Static:         []any{"plugin", name, "user_id", userID, "pid", os.Getpid()},
		Level:          settings.Level,
		RotateMaxBytes: settings.RotateMaxBytes,
		RotateKeep:     settings.RotateKeep,
	})
	if err != nil {
		LogWarn("error at open plugin log files, continuing with stderr only: %v", err)
		return
	}
	fileLogger = slog.Default()
	logEvent(gkill_log.Info, "plugin start",
		"plugin_dir", pluginDir,
		"log_level", gkill_log.LevelName(settings.Level),
		"log_dir", filepath.Join(home, "logs"))
}

// closeLogging は開いたログファイルを閉じる。Run の各終了経路で os.Exit の前に呼ぶ
// （defer は os.Exit で走らない）。何度呼んでも安全。
func closeLogging() {
	if fileLogger == nil {
		return
	}
	fileLogger = nil
	_ = gkill_log.Close()
}

// logToFile は LogWarn / LogError / LogInfo / LogDebug のファイル側。
// source をプラグイン側の呼び出し行にするため Record を自分で組む
// （slog.Logger.Log を経由すると source が log.go になる。MCP の access_log.go と同じ作法）。
func logToFile(level slog.Level, msg string) {
	// runtime.Callers, emit, logToFile, LogXxx の4段を飛ばして呼び出し元
	emit(level, 4, msg)
}

// logEvent は SDK 自身の節目（起動・停止・構築結果・1コマンド1行）用。
func logEvent(level slog.Level, msg string, kv ...any) {
	// runtime.Callers, emit, logEvent の3段を飛ばして呼び出し元（Run / runLoop）
	emit(level, 3, msg, kv...)
}

func emit(level slog.Level, skip int, msg string, kv ...any) {
	logger := fileLogger
	if logger == nil {
		return
	}
	ctx := context.Background()
	handler := logger.Handler()
	if !handler.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(skip, pcs[:])
	record := slog.NewRecord(time.Now(), level, msg, pcs[0])
	record.Add(kv...)
	_ = handler.Handle(ctx, record)
}
