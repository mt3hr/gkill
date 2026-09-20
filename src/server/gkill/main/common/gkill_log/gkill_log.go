// Package gkill_log はレベル別振り分けを行うslogハンドラとログ基盤。
package gkill_log

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// LogLevelFromCmd は --log の値。**既定は error。**
//
// 以前の既定は none で、--log を付けずに起動すると gkill_error.log を含む全ファイルが
// 0バイトのままだった。実際 2026-08-30 の障害では、たまたま --log debug で動かしていた回の
// ログが残っていたおかげでしか原因に辿り着けなかった。「gkill_error.log に出ていなければ
// 起きていない」と言えることを既定にする。
//
// error にしているのは、Warn 以下には常態化しうるもの（rep パターンの0件マッチ、
// プラグインの再起動、認証の失敗）が入るため。回転はあるが、既定で流れ続けさせない。
var LogLevelFromCmd = "error"

var router *Router

// Init は gkill 本体のログ（logs/gkill*.log、app=gkill）を開く。
func Init() {
	InitNamed("gkill", "gkill")
}

// InitNamed は接頭辞 prefix のファイル群（logs/<prefix>.log と <prefix>_<level>.log）を開き、
// 静的フィールド app と extraStatic を全行に付ける。gkill 本体は Init()（prefix "gkill"）、
// MCP サブコマンドは InitNamed("gkill_mcp_<kind>", "gkill_mcp", "kind", kind) で、
// 同じ回転設定・同じレベル語彙のまま別名のファイルへ出す。
// レベルは LogLevelFromCmd（--log）から決める。未知の値は起動を止める。
func InitNamed(prefix, app string, extraStatic ...any) {
	var logLevel slog.Level
	switch strings.ToLower(LogLevelFromCmd) {
	case "trace_sql":
		logLevel = TraceSQL
	case "trace":
		logLevel = Trace
	case "debug":
		logLevel = Debug
	case "access":
		logLevel = Access
	case "info":
		logLevel = Info
	case "warn":
		logLevel = Warn
	case "error":
		logLevel = Error
	case "none":
		logLevel = None
	default:
		log.Fatal("invalid log level. log level [none, error, warn, info, access, debug, trace, trace_sql]")
	}

	logRootDir := os.ExpandEnv(gkill_options.LogDir)
	err := os.MkdirAll(logRootDir, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mkdir %s: %w", logRootDir, err)
		panic(err)
	}

	staticFields := append([]any{"app", app}, extraStatic...)
	router = NewRouter(Options{
		JSON:      true,
		AddSource: true,
		MinLevel:  logLevel,
		// レベル別ファイルと統合ファイルの両方へ出す。
		// SplitOnly だと SetMergedFile で開いた gkill.log へ1バイトも書かれず、
		// 「全レベル統合」と資料に書いてあるファイルが常に空だった。
		Mode:           MergedAndSplit,
		StdoutMirror:   false, //stdoutにも出す
		StaticFields:   staticFields,
		RotateMaxBytes: gkill_options.LogRotateMaxBytes,
		RotateKeep:     gkill_options.LogRotateKeep,
	})

	splitFiles := []struct {
		level  slog.Level
		suffix string
	}{
		{TraceSQL, "_trace_sql.log"},
		{Trace, "_trace.log"},
		{Debug, "_debug.log"},
		{Access, "_access.log"},
		{Info, "_info.log"},
		{Warn, "_warn.log"},
		{Error, "_error.log"},
	}
	for _, split := range splitFiles {
		err = router.SetSplitFile(split.level, filepath.Join(logRootDir, prefix+split.suffix))
		if err != nil {
			panic(err)
		}
	}
	err = router.SetMergedFile(filepath.Join(logRootDir, prefix+".log"))
	if err != nil {
		panic(err)
	}

	slog.SetDefault(router.Logger())
}

// Close は Init / InitNamed が開いたファイルを閉じる（テストと、ログを閉じてから終了したい経路のため）。
func Close() error {
	if router == nil {
		return nil
	}
	return router.Close()
}

func SetMinLevel(level slog.Level) {
	router.SetMinLevel(level)
}

func SetMode(mode SplitMode) {
	router.SetMode(mode)
}

func SetStdoutMirror(isStdoutMirror bool) {
	router.SetStdoutMirror(isStdoutMirror)
}

// Fatal は致命的な失敗を Error で残してからプロセスを終了します。
//
// **標準の log.Fatal を使わないこと。** あれは標準ロガーの stderr へ書くだけなので、
// gkill_error.log には1行も残りません。起動に失敗したサーバでは、あとから原因を
// 見られる場所がそこしかありません（サービスとして動いていると stderr は誰も見ない）。
func Fatal(msg string, err error) {
	if err != nil {
		slog.Log(context.Background(), Error, msg, "error", fmt.Sprintf("%q", err))
	} else {
		slog.Log(context.Background(), Error, msg)
	}
	os.Exit(1)
}
