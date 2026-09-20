package mcp

// GPSログのページングカーソルのコーデック。
//
// **正本はこの1ファイルだけにすること。** 以前は encode/decode が read-handlers に、
// 受け取り側の検証が normalization にあり、後者が gkill_get_kyous 用の複合カーソル
// (`{RFC3339Nano}::{ID}`、Go製) の検証をコピペしたままだった。GPSカーソルは MCP 製の
// base64url なので Date.parse が必ず NaN になり、**説明文どおり next_cursor を verbatim で
// 渡すと 100% `Invalid argument 'cursor'` になっていた**。
// 発行側と検証側が別実装だったことが原因なので、両方からここを使う。

import (
	"encoding/base64"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// EncodeGpsCursor は次ページの開始位置をカーソル文字列にする。
//
// t=最後に返した点の related_time、n=同一時刻の中で消費済みの点数。
// サーバの並びは (related_time降順, latitude昇順, longitude昇順) で決定的なので、
// 同一時刻ランの途中でも位置を特定できる（get_kyous v2 と同じ考え方）。
// 戻り値は base64url(JSON {t, n})（パディング無し。Node の Buffer#toString("base64url") と同じ）。
func EncodeGpsCursor(t string, n int) string {
	payload := jsonobj.MarshalString(jsonobj.Obj("t", t, "n", n))
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

// GpsCursor は復元した位置。
type GpsCursor struct {
	T string
	N int
}

// DecodeGpsCursor はカーソル文字列を {t, n} に戻す。解釈できなければエラー。
func DecodeGpsCursor(cursor string) (GpsCursor, error) {
	if decoded, ok := decodeGpsCursor(cursor); ok {
		return decoded, nil
	}
	return GpsCursor{}, Errorf("Invalid GPS cursor: %s (pass next_cursor verbatim)", jsonobj.MarshalString(cursor))
}

func decodeGpsCursor(cursor string) (GpsCursor, bool) {
	raw, err := decodeBase64URLLenient(cursor)
	if err != nil {
		return GpsCursor{}, false
	}
	v, err := jsonobj.Unmarshal(raw)
	if err != nil {
		return GpsCursor{}, false
	}
	o, ok := v.(*jsonobj.Object)
	if !ok || o == nil {
		return GpsCursor{}, false
	}
	t, ok := o.String("t")
	if !ok {
		return GpsCursor{}, false
	}
	n, ok := o.Int("n")
	if !ok || n < 0 {
		return GpsCursor{}, false
	}
	return GpsCursor{T: t, N: int(n)}, true
}

// decodeBase64URLLenient は Node の Buffer.from(s, "base64url") と同じくパディング有無を問わない。
func decodeBase64URLLenient(s string) ([]byte, error) {
	trimmed := s
	for len(trimmed) > 0 && trimmed[len(trimmed)-1] == '=' {
		trimmed = trimmed[:len(trimmed)-1]
	}
	if out, err := base64.RawURLEncoding.DecodeString(trimmed); err == nil {
		return out, nil
	}
	return base64.RawStdEncoding.DecodeString(trimmed)
}

// IsValidGpsCursor は文字列が GPS カーソルとして解釈できるかを返す。
//
// 入口の検証（normalization）用。ここで弾いておかないと、後段のエラーが
// 「カーソルが原因」だと分からない形に畳まれる。
func IsValidGpsCursor(cursor string) bool {
	_, ok := decodeGpsCursor(cursor)
	return ok
}
