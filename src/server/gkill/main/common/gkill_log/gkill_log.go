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
//
// 開いたあとに、同じレベル・回転設定を環境変数へ書き出す（ExportEnvForChildProcesses）。
// gkill が起動するプラグインの子プロセスはこれを継いで logs/gkill_plugin_<name>*.log を開く
// （GKILL_HOME と同じ経路）。ここで書き出すので、gkill_server / デスクトップ版 /
// generate_plugin_cache のどれからプラグインを起動しても --log が子まで届く。
func Init() {
	InitNamed("gkill", "gkill")
	if err := ExportEnvForChildProcesses(); err != nil {
		slog.Log(context.Background(), Warn, "error at export log settings to child process env", "error", fmt.Sprintf("%q", err))
	}
}

// InitNamed は接頭辞 prefix のファイル群（logs/<prefix>.log と <prefix>_<level>.log）を開き、
// 静的フィールド app と extraStatic を全行に付ける。gkill 本体は Init()（prefix "gkill"）、
// MCP サブコマンドは InitNamed("gkill_mcp_<kind>", "gkill_mcp", "kind", kind) で、
// 同じ回転設定・同じレベル語彙のまま別名のファイルへ出す。
// レベルは LogLevelFromCmd（--log）から決める。未知の値は起動を止める。
func InitNamed(prefix, app string, extraStatic ...any) {
	logLevel, err := ParseLevel(LogLevelFromCmd)
	if err != nil {
		log.Fatal("invalid log level. log level [none, error, warn, info, access, debug, trace, trace_sql]")
	}
	err = InitNamedWith(NamedOptions{
		LogDir:         os.ExpandEnv(gkill_options.LogDir),
		Prefix:         prefix,
		App:            app,
		Static:         extraStatic,
		Level:          logLevel,
		RotateMaxBytes: gkill_options.LogRotateMaxBytes,
		RotateKeep:     gkill_options.LogRotateKeep,
	})
	if err != nil {
		panic(err)
	}
}

// ParseLevel は --log / GKILL_LOG_LEVEL の語彙をレベルにする（大小無視）。
// 語彙は none / error / warn / info / access / debug / trace / trace_sql の8つで、それ以外は error を返す。
func ParseLevel(name string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "trace_sql":
		return TraceSQL, nil
	case "trace":
		return Trace, nil
	case "debug":
		return Debug, nil
	case "access":
		return Access, nil
	case "info":
		return Info, nil
	case "warn":
		return Warn, nil
	case "error":
		return Error, nil
	case "none":
		return None, nil
	default:
		return None, fmt.Errorf("invalid log level %q. log level [none, error, warn, info, access, debug, trace, trace_sql]", name)
	}
}

// NamedOptions は InitNamedWith に渡す、ファイル群1組ぶんの設定。
type NamedOptions struct {
	// LogDir はファイルを置くディレクトリ。無ければ作る。
	LogDir string
	// Prefix はファイル名の接頭辞（<Prefix>.log と <Prefix>_<level>.log）。
	Prefix string
	// App は静的フィールド app の値。
	App string
	// Static は app の後ろに付ける追加の静的フィールド（キーと値の交互）。
	Static []any
	// Level は有効レベルの下限。
	Level slog.Level
	// RotateMaxBytes / RotateKeep は回転設定（Options と同じ意味）。
	RotateMaxBytes int64
	RotateKeep     int
}

// InitNamedWith は InitNamed の本体。LogLevelFromCmd や gkill_options を見ず、渡された設定だけで開く。
//
// InitNamed（本体・MCP）はディレクトリが作れなければ panic するが、こちらは error を返す。
// プラグインの子プロセスはログの都合で止めたくない（ログ dir が無いのは手起動か権限の問題で、
// 記録の読み取りとは無関係）ので、失敗を受け取って stderr だけで続行する。
// 失敗したときは開いた sink を閉じ、slog の既定ロガーも package の router も差し替えない。
func InitNamedWith(o NamedOptions) error {
	if err := os.MkdirAll(o.LogDir, os.ModePerm); err != nil {
		return fmt.Errorf("error at mkdir %s: %w", o.LogDir, err)
	}

	staticFields := append([]any{"app", o.App}, o.Static...)
	newRouter := NewRouter(Options{
		JSON:      true,
		AddSource: true,
		MinLevel:  o.Level,
		// レベル別ファイルと統合ファイルの両方へ出す。
		// SplitOnly だと SetMergedFile で開いた gkill.log へ1バイトも書かれず、
		// 「全レベル統合」と資料に書いてあるファイルが常に空だった。
		Mode:           MergedAndSplit,
		StdoutMirror:   false, //stdoutにも出す
		StaticFields:   staticFields,
		RotateMaxBytes: o.RotateMaxBytes,
		RotateKeep:     o.RotateKeep,
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
		if err := newRouter.SetSplitFile(split.level, filepath.Join(o.LogDir, o.Prefix+split.suffix)); err != nil {
			_ = newRouter.Close()
			return fmt.Errorf("error at open split log file %s%s: %w", o.Prefix, split.suffix, err)
		}
	}
	if err := newRouter.SetMergedFile(filepath.Join(o.LogDir, o.Prefix+".log")); err != nil {
		_ = newRouter.Close()
		return fmt.Errorf("error at open merged log file %s.log: %w", o.Prefix, err)
	}

	router = newRouter
	slog.SetDefault(router.Logger())
	return nil
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
