package mcp

// JavaScript の値意味論を Go で再現する小道具。
//
// Node 実装の応答をバイト単位で一致させるため、テンプレート文字列（String(v)）と
// truthiness（`if (value)`）の規則をそのまま持つ。

import (
	"math"
	"runtime"
	"strconv"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// jsString は JS の String(v)（テンプレート文字列の埋め込みと同じ）。
func jsString(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case jsonobj.UndefinedType:
		return "undefined"
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case *jsonobj.Object:
		if x == nil {
			return "null"
		}
		return "[object Object]"
	case error:
		return x.Error()
	}
	if f, ok := jsonobj.ToFloat(v); ok {
		return jsNumberString(f)
	}
	if arr, ok := jsonobj.AsArray(v); ok {
		parts := make([]string, len(arr))
		for i, item := range arr {
			if item == nil || jsonobj.IsUndefined(item) {
				parts[i] = ""
				continue
			}
			parts[i] = jsString(item)
		}
		return strings.Join(parts, ",")
	}
	return jsonobj.MarshalString(v)
}

// jsNumberString は JS の Number#toString（ES6 の Number::toString）。
func jsNumberString(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		return "0"
	}
	return jsonobj.MarshalString(f)
}

// jsTruthy は JS の truthiness（`if (value)`）。
func jsTruthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case jsonobj.UndefinedType:
		return false
	case string:
		return x != ""
	case bool:
		return x
	case *jsonobj.Object:
		return x != nil
	}
	if f, ok := jsonobj.ToFloat(v); ok {
		return f != 0 && !math.IsNaN(f)
	}
	return true
}

// itoa は strconv.Itoa の短い別名。
func itoa(n int) string { return strconv.Itoa(n) }

// isWindows は実行環境が Windows か（ファイルのモードビットの検査を飛ばす判断に使う）。
func isWindows() bool { return runtime.GOOS == "windows" }
