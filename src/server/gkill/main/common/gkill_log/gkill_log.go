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

var LogLevelFromCmd = "none"

var router *Router

func Init() {
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

	router = NewRouter(Options{
		JSON:      true,
		AddSource: true,
		MinLevel:  logLevel,
		// レベル別ファイルと統合ファイルの両方へ出す。
		// SplitOnly だと SetMergedFile で開いた gkill.log へ1バイトも書かれず、
		// 「全レベル統合」と資料に書いてあるファイルが常に空だった。
		Mode:           MergedAndSplit,
		StdoutMirror:   false, //stdoutにも出す
		StaticFields:   []any{"app", "gkill"},
		RotateMaxBytes: gkill_options.LogRotateMaxBytes,
		RotateKeep:     gkill_options.LogRotateKeep,
	})

	err = router.SetSplitFile(TraceSQL, filepath.Join(logRootDir, "gkill_trace_sql.log"))
	if err != nil {
		panic(err)
	}
	err = router.SetSplitFile(Trace, filepath.Join(logRootDir, "gkill_trace.log"))
	if err != nil {
		panic(err)
	}
	err = router.SetSplitFile(Debug, filepath.Join(logRootDir, "gkill_debug.log"))
	if err != nil {
		panic(err)
	}
	err = router.SetSplitFile(Access, filepath.Join(logRootDir, "gkill_access.log"))
	if err != nil {
		panic(err)
	}
	err = router.SetSplitFile(Info, filepath.Join(logRootDir, "gkill_info.log"))
	if err != nil {
		panic(err)
	}
	err = router.SetSplitFile(Warn, filepath.Join(logRootDir, "gkill_warn.log"))
	if err != nil {
		panic(err)
	}
	err = router.SetSplitFile(Error, filepath.Join(logRootDir, "gkill_error.log"))
	if err != nil {
		panic(err)
	}
	err = router.SetMergedFile(filepath.Join(logRootDir, "gkill.log"))
	if err != nil {
		panic(err)
	}

	slog.SetDefault(router.Logger())
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
