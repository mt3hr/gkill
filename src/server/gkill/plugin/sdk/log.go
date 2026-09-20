package sdk

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// プラグインのログは2つの出口へ出す。
//
//  1. stderr: `WARN: ` / `ERROR: ` の接頭辞を付けた1行。gkill 本体は stderr の末尾を
//     pluginStderrRing に溜めて PluginInfo.last_error として見せる（人間が読む診断）。
//  2. gkill_log: $GKILL_HOME/logs/gkill_plugin_<name>*.log（JSON、本体と同じレベル語彙・回転）。
//     Run が initLogging で開く。Info / Debug はこちらにだけ出す（stderr のリングは 4KB しか
//     無いので、節目の行で肝心のエラーを押し出さない）。
//
// **os.Stdout には絶対に書かないこと。** あれはプロトコルのチャネルで、1行でも混ざると
// JSON ストリームが壊れる。
//
// 接頭辞を付けるのは、以前はスキップ（1ファイル読めなかっただけで続行する）と
// 構築全体の失敗が同じ見た目で同じストリームに混ざっていて、grep でも
// 深刻度を区別できなかったため。
const (
	logPrefixWarn  = "WARN: "
	logPrefixError = "ERROR: "
)

// logWriter は差し替え可能な stderr 側の出力先（テスト用）。
var logWriter io.Writer = os.Stderr

// SetLogWriter は stderr 側の出力先を差し替えます。テスト以外では呼ばないこと。
// gkill_log 側のファイルには影響しない。
func SetLogWriter(w io.Writer) {
	if w == nil {
		logWriter = os.Stderr
		return
	}
	logWriter = w
}

// LogWarn は「続行できるが結果が痩せる」事象を出します。
// 読めなかった1ファイルのスキップ、壊れた1レコードの読み飛ばしなど。
// stderr（last_error）とログファイルの両方へ出ます。
func LogWarn(format string, args ...any) {
	logWithStderr(gkill_log.Warn, logPrefixWarn, format, args...)
}

// LogError は「その処理が失敗した」事象を出します。
// 取り込み全体の失敗、manifest/config の書き出し失敗など。
// stderr（last_error）とログファイルの両方へ出ます。
func LogError(format string, args ...any) {
	logWithStderr(gkill_log.Error, logPrefixError, format, args...)
}

// LogInfo は節目（構築完了・取り込み元の切り替えなど。1事象1行で流れ続けないもの）を出します。
// ログファイルにだけ出ます（--log info 以上で残る）。stderr には出ません。
func LogInfo(format string, args ...any) { logToFile(gkill_log.Info, fmt.Sprintf(format, args...)) }

// LogDebug は開発時の詳細を出します。ログファイルにだけ出ます（--log debug 以上で残る）。
// エラーの置き場ではありません（ADR-1001）。失敗は LogWarn / LogError へ。
// 1行関数のままにすること: 複数行にすると log_level_source_scan_test の
// 「ブロック末尾の Debug ＝握り潰し」検査がこの定義を誤検知する。
func LogDebug(format string, args ...any) { logToFile(gkill_log.Debug, fmt.Sprintf(format, args...)) }

// logWithStderr は stderr の接頭辞行とログファイルの両方へ出す（Warn / Error）。
// 呼び出しの深さは logToFile と揃えてある（LogXxx → ここ → emit）。source の skip が同じ数で済む。
func logWithStderr(level slog.Level, prefix string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(logWriter, prefix+msg)
	// runtime.Callers, emit, logWithStderr, LogXxx の4段を飛ばして呼び出し元
	emit(level, 4, msg)
}
