package kftl

import (
	"testing"
	"time"
)

// TestParseDateTime は時刻のみ/年省略/完全日時の補完を固定 base で検査する。
// クライアント側 src/client/__tests__/unit/kftl/kftl-date-time.test.ts と対。
func TestParseDateTime(t *testing.T) {
	base := time.Date(2026, 8, 20, 21, 30, 0, 0, time.Local)

	cases := []struct {
		name  string
		input string
		want  time.Time
	}{
		{"時刻のみ HH:MM は base の年月日に載る", "15:04", time.Date(2026, 8, 20, 15, 4, 0, 0, time.Local)},
		{"時刻のみ HH:MM:SS", "15:04:05", time.Date(2026, 8, 20, 15, 4, 5, 0, time.Local)},
		{"年省略 M/D HH:MM は月日を尊重し年だけ補完", "1/2 15:04", time.Date(2026, 1, 2, 15, 4, 0, 0, time.Local)},
		{"年省略 MM/DD HH:MM", "01/02 15:04", time.Date(2026, 1, 2, 15, 4, 0, 0, time.Local)},
		{"完全日時はそのまま", "2026-03-15 10:20:30", time.Date(2026, 3, 15, 10, 20, 30, 0, time.Local)},
		{"日付のみは 00:00", "2026-03-15", time.Date(2026, 3, 15, 0, 0, 0, 0, time.Local)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseDateTime(c.input, base)
			if err != nil {
				t.Fatalf("parseDateTime(%q) unexpected error: %v", c.input, err)
			}
			if !got.Equal(c.want) {
				t.Errorf("parseDateTime(%q) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}

// 年またぎ境界: 時刻のみ入力は「前日」推測をせず base と同じ日に補完する（現仕様の固定）。
func TestParseDateTime_YearBoundaryDoesNotInferPreviousDay(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 10, 0, 0, time.Local)
	got, err := parseDateTime("23:50", base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 1, 1, 23, 50, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("parseDateTime(\"23:50\") = %v, want %v", got, want)
	}
}

// パースできない入力はエラー。
func TestParseDateTime_Invalid(t *testing.T) {
	base := time.Date(2026, 8, 20, 21, 30, 0, 0, time.Local)
	if _, err := parseDateTime("not a time", base); err == nil {
		t.Error("expected error for unparseable input")
	}
}
