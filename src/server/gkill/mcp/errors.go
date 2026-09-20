// Package mcp は gkill の MCP サーバ（read / write / readwrite の3種、stdio + HTTP OAuth 2.1）。
//
// 2026-09-20 に Node.js 実装（旧 src/mcp、最終コミットは ADR-0631 参照）を Go へ移したもので、
// ファイルは旧 lib/*.mjs と 1:1 に対応する（errors.go ← errors.mjs …）。
// 応答のバイト列を旧実装と一致させるため、値は jsonobj（順序つき JSON）で持ち回る。
//
// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
package mcp

import (
	"errors"
	"fmt"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// GkillApiError は MCP 層の検証・正規化・gkill 呼び出しの失敗を表す（旧 errors.mjs の GkillApiError）。
//
// Detail は呼び出し側へ機械可読に返す補足（field / actualType / allowed …）。
// detail.field が string のときは「呼び出し側の誤り」（HTTP でいう 400）として扱われる。
type GkillApiError struct {
	Message string
	Detail  *jsonobj.Object
}

func (e *GkillApiError) Error() string {
	return e.Message
}

// NewGkillApiError は detail 付きのエラーを作る。detail は nil / *jsonobj.Object のほか、
// gkill の応答（*jsonobj.Object）や Go の struct（jsonobj.FromGo で写す）を受ける。
func NewGkillApiError(message string, detail any) *GkillApiError {
	return &GkillApiError{Message: message, Detail: toDetailObject(detail)}
}

// Errorf は detail 無しの GkillApiError。
func Errorf(format string, args ...any) *GkillApiError {
	return &GkillApiError{Message: fmt.Sprintf(format, args...)}
}

func toDetailObject(detail any) *jsonobj.Object {
	switch d := detail.(type) {
	case nil:
		return nil
	case *jsonobj.Object:
		return d
	default:
		converted, err := jsonobj.FromGo(detail)
		if err != nil {
			return jsonobj.Obj("detail", fmt.Sprintf("%v", detail))
		}
		if o, ok := converted.(*jsonobj.Object); ok {
			return o
		}
		return jsonobj.Obj("detail", converted)
	}
}

// AsGkillApiError は err が GkillApiError なら取り出す。
func AsGkillApiError(err error) (*GkillApiError, bool) {
	var target *GkillApiError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// DetailField は detail.field が string ならそれを返す（呼び出し側の誤りの判定に使う）。
func DetailField(err error) (string, bool) {
	apiErr, ok := AsGkillApiError(err)
	if !ok || apiErr.Detail == nil {
		return "", false
	}
	return apiErr.Detail.String("field")
}

// IsPlainObject は値が JS の「素のオブジェクト」（null でも配列でもないオブジェクト）か。
func IsPlainObject(value any) bool {
	o, ok := value.(*jsonobj.Object)
	return ok && o != nil
}

// DescribeValueType は値の種別名（null / array / object / string / number / boolean / undefined）。
func DescribeValueType(value any) string {
	return jsonobj.TypeOf(value)
}

// PreviewValue はエラーの detail.actualValue に載せる短い表現。
func PreviewValue(value any) any {
	switch v := value.(type) {
	case string:
		runes := []rune(v)
		if len(runes) > 120 {
			return string(runes[:117]) + "..."
		}
		return v
	case bool:
		return v
	case nil:
		return nil
	}
	if jsonobj.IsNumber(value) {
		return value
	}
	if arr, ok := jsonobj.AsArray(value); ok {
		return fmt.Sprintf("array(%d)", len(arr))
	}
	if IsPlainObject(value) {
		return "object"
	}
	return DescribeValueType(value)
}

// InvalidArgument は「引数 field が message」のエラー。extra はキーと値を交互に並べた追加の detail。
func InvalidArgument(field, message string, value any, extra ...any) *GkillApiError {
	detail := jsonobj.Obj(
		"field", field,
		"actualType", DescribeValueType(value),
		"actualValue", PreviewValue(value),
	)
	if len(extra)%2 != 0 {
		panic("mcp.InvalidArgument: extra must be key/value pairs")
	}
	for i := 0; i < len(extra); i += 2 {
		key, ok := extra[i].(string)
		if !ok {
			panic("mcp.InvalidArgument: extra key is not a string")
		}
		detail.Set(key, extra[i+1])
	}
	return &GkillApiError{
		Message: fmt.Sprintf("Invalid argument '%s': %s.", field, message),
		Detail:  detail,
	}
}
