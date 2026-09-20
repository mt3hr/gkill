package mcp

// JS の Date.parse / Date#toISOString に相当する最小の互換層。
//
// 旧実装（Node）は日時の比較・検証に Date.parse を使っていた。RFC 3339 の入力に対しては
// Go の time.Parse と同じ答えになる。Date.parse が受理する RFC 3339 以外の形（"Aug 24 2026" 等）は
// 呼び出し側の入口で RFC 3339 か日付だけに絞られているので、ここでは ISO 8601 の派生形だけを受ける。

import (
	"strings"
	"time"
)

// jsDateLayouts は Date.parse が受ける ISO 8601 の形（オフセット無しはローカル時刻、日付だけは UTC）。
var jsDateLayoutsWithZone = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04Z07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05Z07:00",
}

var jsDateLayoutsLocal = []string{
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
}

// jsDateParse は Date.parse。解釈できなければ false（NaN）。
func jsDateParse(value string) (time.Time, bool) {
	s := strings.TrimSpace(value)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range jsDateLayoutsWithZone {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	if len(s) == len("2006-01-02") {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t, true
		}
	}
	for _, layout := range jsDateLayoutsLocal {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// jsDateParseMillis は Date.parse のミリ秒値（NaN は false）。
func jsDateParseMillis(value string) (int64, bool) {
	t, ok := jsDateParse(value)
	if !ok {
		return 0, false
	}
	return t.UnixMilli(), true
}

// padStart は String#padStart（左詰め）。
func padStart(s string, width int, pad string) string {
	for jsLength(s) < width {
		s = pad + s
	}
	if jsLength(s) > width && len(pad) > 1 {
		// pad が複数文字のとき、余分は先頭側を切る（padStart の規則）
		over := jsLength(s) - width
		s = jsSlice(s, over, jsLength(s))
	}
	return s
}

// padEnd は String#padEnd（右詰め。pad が複数文字なら末尾を切る）。
func padEnd(s string, width int, pad string) string {
	if pad == "" {
		return s
	}
	for jsLength(s) < width {
		s += pad
	}
	if jsLength(s) > width {
		s = jsSlice(s, 0, width)
	}
	return s
}
