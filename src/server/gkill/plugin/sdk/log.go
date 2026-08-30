package sdk

import (
	"fmt"
	"io"
	"os"
)

// プラグインのログはレベルを接頭辞で表す。
//
// **os.Stdout には絶対に書かないこと。** あれはプロトコルのチャネルで、1行でも混ざると
// JSON ストリームが壊れる。ログは必ず stderr へ出す（本体側は stderr の末尾を
// pluginStderrRing に溜めて PluginInfo.last_error として見せる）。
//
// 接頭辞を付けるのは、以前はスキップ（1ファイル読めなかっただけで続行する）と
// 構築全体の失敗が同じ見た目で同じストリームに混ざっていて、grep でも
// 深刻度を区別できなかったため。
const (
	logPrefixWarn  = "WARN: "
	logPrefixError = "ERROR: "
)

// logWriter は差し替え可能なログの出力先（テスト用）。
var logWriter io.Writer = os.Stderr

// SetLogWriter はログの出力先を差し替えます。テスト以外では呼ばないこと。
func SetLogWriter(w io.Writer) {
	if w == nil {
		logWriter = os.Stderr
		return
	}
	logWriter = w
}

// LogWarn は「続行できるが結果が痩せる」事象を出します。
// 読めなかった1ファイルのスキップ、壊れた1レコードの読み飛ばしなど。
func LogWarn(format string, args ...any) {
	fmt.Fprintf(logWriter, logPrefixWarn+format+"\n", args...)
}

// LogError は「その処理が失敗した」事象を出します。
// 取り込み全体の失敗、manifest/config の書き出し失敗など。
func LogError(format string, args ...any) {
	fmt.Fprintf(logWriter, logPrefixError+format+"\n", args...)
}
