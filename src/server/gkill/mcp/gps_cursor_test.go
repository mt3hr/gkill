package mcp

import (
	"encoding/base64"
	"testing"
)

// GPS カーソルは発行側（EncodeGpsCursor）と受け取り側（IsValidGpsCursor / DecodeGpsCursor）を
// この1ファイルで揃える。旧実装は検証だけが別のコピーで、next_cursor を verbatim で渡すと 100% 拒否されていた。
func TestGpsCursorRoundTripAndValidation(t *testing.T) {
	t.Run("encode → decode restores t and n; validation agrees", func(t *testing.T) {
		cursor := EncodeGpsCursor("2026-09-20T18:30:45.123+09:00", 3)
		expectTrue(t, IsValidGpsCursor(cursor), "own cursor rejected: %q", cursor)
		decoded, err := DecodeGpsCursor(cursor)
		expectNoError(t, err)
		expectEqual(t, decoded.T, "2026-09-20T18:30:45.123+09:00")
		expectEqual(t, decoded.N, 3)
	})

	t.Run("padding is optional (Node's base64url and padded base64 both decode)", func(t *testing.T) {
		raw := []byte(`{"t":"2026-01-01T00:00:00+09:00","n":0}`)
		padded := base64.URLEncoding.EncodeToString(raw)
		std := base64.StdEncoding.EncodeToString(raw)
		expectTrue(t, IsValidGpsCursor(padded), "padded base64url rejected")
		expectTrue(t, IsValidGpsCursor(std), "standard base64 rejected")
	})

	t.Run("rejects garbage, non-object JSON, missing t, negative n", func(t *testing.T) {
		encode := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
		for name, cursor := range map[string]string{
			"not base64":       "%%%",
			"not json":         encode("hello"),
			"array":            encode(`[1]`),
			"missing t":        encode(`{"n":1}`),
			"t is not string":  encode(`{"t":1,"n":1}`),
			"negative n":       encode(`{"t":"x","n":-1}`),
			"n is not integer": encode(`{"t":"x","n":"1"}`),
			"empty":            "",
		} {
			expectTrue(t, !IsValidGpsCursor(cursor), "%s accepted: %q", name, cursor)
			_, err := DecodeGpsCursor(cursor)
			expectErrorContains(t, err, "Invalid GPS cursor")
		}
	})
}
