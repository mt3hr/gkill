package gkill_log

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

var router *Router

func init() {
	logRootDir := os.ExpandEnv(gkill_options.LogDir)
	err := os.MkdirAll(logRootDir, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mkdir %s: %w", logRootDir, err)
		panic(err)
	}

	router = NewRouter(Options{
		JSON:         true,
		AddSource:    true,
		MinLevel:     None,
		Mode:         SplitOnly, // ログファイルをまとめるやつ
		StdoutMirror: false,     //stdoutにも出す
		StaticFields: []any{"app", "gkill"},
	})

	router.SetSplitFile(TraceSQL, filepath.Join(logRootDir, "gkill_trace_sql.log"))
	router.SetSplitFile(Trace, filepath.Join(logRootDir, "gkill_trace.log"))
	router.SetSplitFile(Debug, filepath.Join(logRootDir, "gkill_debug.log"))
	router.SetSplitFile(Info, filepath.Join(logRootDir, "gkill_info.log"))
	router.SetSplitFile(Warn, filepath.Join(logRootDir, "gkill_warn.log"))
	router.SetSplitFile(Error, filepath.Join(logRootDir, "gkill_error.log"))
	router.SetMergedFile(filepath.Join(logRootDir, "gkill.log"))

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
