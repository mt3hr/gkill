package gkill_log

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// gkill が起動する子プロセス（プラグイン）へログの設定を渡す環境変数。
//
// フラグ（--gkill-log-level など）にしなかったのは、gkill の SDK を使わない第三者のプラグインが
// 標準の flag で未知のフラグに exit 2 で落ちるため（generate_plugin_cache が「旧バイナリ」と
// 数える形。gkill-cli-ops スキル）。環境変数なら知らないプラグインは無視するだけで済む。
// GKILL_HOME（main/common/common.go の InitGkillOptions）と同じ経路。
const (
	// EnvLogLevel は --log の値（none / error / warn / info / access / debug / trace / trace_sql）。
	EnvLogLevel = "GKILL_LOG_LEVEL"
	// EnvLogRotateMaxBytes は --log_rotate_max_bytes の値。
	EnvLogRotateMaxBytes = "GKILL_LOG_ROTATE_MAX_BYTES"
	// EnvLogRotateKeep は --log_rotate_keep の値。
	EnvLogRotateKeep = "GKILL_LOG_ROTATE_KEEP"
)

// ExportEnvForChildProcesses は現在のレベル（LogLevelFromCmd）と回転設定（gkill_options）を
// 環境変数へ書き出す。exec.Cmd は Env が nil なら os.Environ() を継ぐので、
// これ以後に起動した子プロセスすべてに届く。Init() が呼ぶ。
func ExportEnvForChildProcesses() error {
	if err := os.Setenv(EnvLogLevel, LogLevelFromCmd); err != nil {
		return fmt.Errorf("error at setenv %s: %w", EnvLogLevel, err)
	}
	if err := os.Setenv(EnvLogRotateMaxBytes, strconv.FormatInt(gkill_options.LogRotateMaxBytes, 10)); err != nil {
		return fmt.Errorf("error at setenv %s: %w", EnvLogRotateMaxBytes, err)
	}
	if err := os.Setenv(EnvLogRotateKeep, strconv.Itoa(gkill_options.LogRotateKeep)); err != nil {
		return fmt.Errorf("error at setenv %s: %w", EnvLogRotateKeep, err)
	}
	return nil
}

// ChildSettings は子プロセスが環境変数から読み取ったログの設定。
type ChildSettings struct {
	Level          slog.Level
	RotateMaxBytes int64
	RotateKeep     int
	// Warnings は壊れていて既定へ倒した値の説明（人間向け）。子はこれを stderr に出す。
	Warnings []string
}

// ChildSettingsFromEnv は ExportEnvForChildProcesses が書いた環境変数を読む。
//
// 無い・壊れている値は既定（error / gkill_options の回転既定値）へ倒し、Warnings で知らせる。
// **子プロセスを止めない。** レベル名の打ち間違いは MCP なら起動を止めるが、
// プラグインは設定される側で、親（gkill）は自分で検証した値しか渡さない。
// 手起動で壊れた値を入れたときだけ Warnings が出る。
// 既定値を gkill_options の変数から取るのは、子プロセスではフラグを解析しないので
// その変数が既定値のままだから（親と同じ既定を1箇所で持つ）。
func ChildSettingsFromEnv() ChildSettings {
	settings := ChildSettings{
		Level:          Error,
		RotateMaxBytes: gkill_options.LogRotateMaxBytes,
		RotateKeep:     gkill_options.LogRotateKeep,
	}
	if raw, ok := os.LookupEnv(EnvLogLevel); ok && raw != "" {
		level, err := ParseLevel(raw)
		if err != nil {
			settings.Warnings = append(settings.Warnings, fmt.Sprintf("%s=%q is not a log level, using error", EnvLogLevel, raw))
		} else {
			settings.Level = level
		}
	}
	if raw, ok := os.LookupEnv(EnvLogRotateMaxBytes); ok && raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			settings.Warnings = append(settings.Warnings, fmt.Sprintf("%s=%q is not an integer, using %d", EnvLogRotateMaxBytes, raw, settings.RotateMaxBytes))
		} else {
			settings.RotateMaxBytes = value
		}
	}
	if raw, ok := os.LookupEnv(EnvLogRotateKeep); ok && raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			settings.Warnings = append(settings.Warnings, fmt.Sprintf("%s=%q is not an integer, using %d", EnvLogRotateKeep, raw, settings.RotateKeep))
		} else {
			settings.RotateKeep = value
		}
	}
	return settings
}
