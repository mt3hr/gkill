package gkill_log

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

var LogLevelFromCmd = "none"

var router *Router

func Init() {
	logLevel := None
	switch strings.ToLower(LogLevelFromCmd) {
	case "trace_sql":
		logLevel = TraceSQL
	case "trace":
		logLevel = Trace
	case "debug":
		logLevel = Debug
	case "info":
		logLevel = Info
	case "warn":
		logLevel = Warn
	case "error":
		logLevel = Error
	case "none":
		logLevel = None
	default:
		log.Fatal("invalid log level. log level [none, error, warn, info, debug, trace, trace_sql]")
	}

	logRootDir := os.ExpandEnv(gkill_options.LogDir)
	err := os.MkdirAll(logRootDir, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mkdir %s: %w", logRootDir, err)
		panic(err)
	}

	router = NewRouter(Options{
		JSON:         true,
		AddSource:    true,
		MinLevel:     logLevel,
		Mode:         SplitOnly, // ログファイルをまとめるやつ
		StdoutMirror: false,     //stdoutにも出す
		StaticFields: []any{"app", "gkill"},
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
	router.SetStdoutMirror(true)
}
