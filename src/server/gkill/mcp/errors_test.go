package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// InvalidArgument が detail に載せる actualType / actualValue の形。
// ゴールデンは detail.cause をマスクするので、ここで直接固定する（クライアントは actualType で
// 「型が違う」と「値が範囲外」を見分ける）。
func TestInvalidArgumentDetailShape(t *testing.T) {
	t.Run("string values are previewed as-is and long ones are cut at 120 runes", func(t *testing.T) {
		expectEqual(t, PreviewValue("short"), "short")
		long := strings.Repeat("あ", 130)
		preview := PreviewValue(long).(string)
		expectEqual(t, len([]rune(preview)), 120)
		expectTrue(t, strings.HasSuffix(preview, "..."), "long preview is not cut with ...: %q", preview)
		expectEqual(t, DescribeValueType("s"), "string")
	})

	t.Run("numbers, booleans and null pass through", func(t *testing.T) {
		expectEqual(t, PreviewValue(true), true)
		expectEqual(t, PreviewValue(nil), nil)
		expectEqual(t, DescribeValueType(nil), "null")
		expectEqual(t, DescribeValueType(true), "boolean")
		n := PreviewValue(json.Number("12"))
		expectTrue(t, jsonobj.IsNumber(n), "number preview lost its type: %#v", n)
		expectEqual(t, DescribeValueType(json.Number("12")), "number")
	})

	t.Run("arrays and objects are summarized, not dumped", func(t *testing.T) {
		expectEqual(t, PreviewValue([]any{1, 2, 3}), "array(3)")
		expectEqual(t, DescribeValueType([]any{}), "array")
		expectEqual(t, PreviewValue(jsonobj.Obj("a", 1)), "object")
		expectEqual(t, DescribeValueType(jsonobj.Obj("a", 1)), "object")
		expectTrue(t, IsPlainObject(jsonobj.Obj()), "empty object is not plain")
		expectTrue(t, !IsPlainObject([]any{}), "array counted as plain object")
		expectTrue(t, !IsPlainObject(nil), "nil counted as plain object")
		var nilObject *jsonobj.Object
		expectTrue(t, !IsPlainObject(nilObject), "nil *Object counted as plain object")
	})

	t.Run("InvalidArgument builds the message and detail with extra key/value pairs", func(t *testing.T) {
		err := InvalidArgument("limit", "must be between 1 and 1000", []any{1}, "allowed", jsonobj.Strings("a", "b"))
		expectEqual(t, err.Message, "Invalid argument 'limit': must be between 1 and 1000.")
		field, _ := err.Detail.String("field")
		expectEqual(t, field, "limit")
		actualType, _ := err.Detail.String("actualType")
		expectEqual(t, actualType, "array")
		actualValue, _ := err.Detail.String("actualValue")
		expectEqual(t, actualValue, "array(1)")
		allowed, ok := jsonobj.AsArray(err.Detail.Value("allowed"))
		expectTrue(t, ok && len(allowed) == 2, "extra detail not set: %v", err.Detail.Value("allowed"))
	})

	t.Run("odd extra arguments are a programming error and panic", func(t *testing.T) {
		defer func() {
			expectTrue(t, recover() != nil, "odd extra did not panic")
		}()
		InvalidArgument("x", "m", nil, "dangling")
	})
}
