package main

import (
	"runtime"
	"testing"
	"time"
)

func TestArgsStructFields(t *testing.T) {
	a := Args{
		FitbitPath:   "/path/to/fitbit",
		KCDBPath:     "/path/to/kc.db",
		TagDBPath:    "/path/to/tag.db",
		User:         "testuser",
		Device:       "PixelWatch2",
		SourceTZ:     "Asia/Tokyo",
		ParseWorkers: 4,
	}
	if a.FitbitPath != "/path/to/fitbit" {
		t.Errorf("FitbitPath = %q", a.FitbitPath)
	}
	if a.Device != "PixelWatch2" {
		t.Errorf("Device = %q", a.Device)
	}
}

func TestDecideTimeParser_RFC3339(t *testing.T) {
	samples := []string{"2024-01-15T10:30:00+09:00"}
	tp := decideTimeParser(samples)
	if tp.layout != time.RFC3339 {
		t.Errorf("layout = %q, want RFC3339", tp.layout)
	}
	if tp.naive {
		t.Error("expected naive=false for RFC3339")
	}
}

func TestDecideTimeParser_SlashDate(t *testing.T) {
	samples := []string{"01/15/2024 10:30"}
	tp := decideTimeParser(samples)
	if tp.layout != "01/02/2006 15:04:05" {
		t.Errorf("layout = %q, want slash date layout", tp.layout)
	}
	if !tp.naive {
		t.Error("expected naive=true for slash date")
	}
	if !tp.addSeconds {
		t.Error("expected addSeconds=true for HH:MM format")
	}
}

func TestDecideTimeParser_DashDate(t *testing.T) {
	samples := []string{"2024-01-15 10:30:45"}
	tp := decideTimeParser(samples)
	if tp.layout != "2006-01-02 15:04:05" {
		t.Errorf("layout = %q, want dash date layout", tp.layout)
	}
	if !tp.naive {
		t.Error("expected naive=true for dash date")
	}
}

func TestDecideTimeParser_DateOnly(t *testing.T) {
	samples := []string{"2024-01-15"}
	tp := decideTimeParser(samples)
	if tp.layout != "2006-01-02" {
		t.Errorf("layout = %q, want date-only layout", tp.layout)
	}
}

func TestDecideTimeParser_Empty(t *testing.T) {
	tp := decideTimeParser(nil)
	if tp.layout != time.RFC3339 {
		t.Errorf("fallback layout = %q, want RFC3339", tp.layout)
	}
}

func TestParseWithTP_RFC3339(t *testing.T) {
	tp := timeParser{layout: time.RFC3339, naive: false}
	got, err := parseWithTP("2024-01-15T10:30:00Z", "Asia/Tokyo", tp)
	if err != nil {
		t.Fatalf("parseWithTP: %v", err)
	}
	if got.Year() != 2024 || got.Month() != 1 || got.Day() != 15 {
		t.Errorf("unexpected date: %v", got)
	}
}

func TestParseWithTP_Empty(t *testing.T) {
	tp := timeParser{layout: time.RFC3339}
	_, err := parseWithTP("", "Asia/Tokyo", tp)
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestNormHeader(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Beats per minute", "beatsperminute"},
		{" Heart Rate ", "heartrate"},
		{"time_stamp", "timestamp"},
	}
	for _, tt := range tests {
		got := normHeader(tt.input)
		if got != tt.want {
			t.Errorf("normHeader(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFindCol(t *testing.T) {
	headers := []string{"Timestamp", "Beats per minute", "Extra"}
	idx := findCol(headers, "beatsperminute", "value")
	if idx != 1 {
		t.Errorf("findCol = %d, want 1", idx)
	}

	idx = findCol(headers, "nonexistent")
	if idx != -1 {
		t.Errorf("findCol for nonexistent = %d, want -1", idx)
	}
}

func TestDetectMetric_HeartRate(t *testing.T) {
	md, ok := detectMetric("fitbit/heart_rate-2024-01-15.csv")
	if !ok {
		t.Fatal("expected match for heart rate CSV")
	}
	if md.Key != "heart_rate" {
		t.Errorf("Key = %q, want heart_rate", md.Key)
	}
}

func TestDetectMetric_Steps(t *testing.T) {
	md, ok := detectMetric("fitbit/minute_steps-2024-01.csv")
	if !ok {
		t.Fatal("expected match for steps CSV")
	}
	if md.Key != "steps" {
		t.Errorf("Key = %q, want steps", md.Key)
	}
}

func TestDetectMetric_Calories(t *testing.T) {
	md, ok := detectMetric("fitbit/minute_calories-2024-01.csv")
	if !ok {
		t.Fatal("expected match for calories CSV")
	}
	if md.Key != "calories" {
		t.Errorf("Key = %q, want calories", md.Key)
	}
}

func TestDetectMetric_HRV_Excluded(t *testing.T) {
	_, ok := detectMetric("fitbit/hrv_heart_rate-2024-01.csv")
	// HRV paths should be excluded from heart rate detection
	if ok {
		t.Error("expected no match for HRV path")
	}
}

func TestDetectMetric_Unknown(t *testing.T) {
	_, ok := detectMetric("fitbit/sleep-2024-01.csv")
	if ok {
		t.Error("expected no match for unknown metric")
	}
}

func TestFmtUTC00(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	got := fmtUTC00(ts)
	want := "2024-01-15T10:30:00+00:00"
	if got != want {
		t.Errorf("fmtUTC00 = %q, want %q", got, want)
	}
}

func TestFallbackValueIndex(t *testing.T) {
	// Headers with bpm
	idx := fallbackValueIndex([]string{"Time", "bpm", "extra"})
	if idx != 1 {
		t.Errorf("fallbackValueIndex with bpm = %d, want 1", idx)
	}

	// Headers without keywords -> fallback to index 1
	idx = fallbackValueIndex([]string{"col0", "col1", "col2"})
	if idx != 1 {
		t.Errorf("fallbackValueIndex no keywords = %d, want 1", idx)
	}

	// Single header
	idx = fallbackValueIndex([]string{"only"})
	if idx != -1 {
		t.Errorf("fallbackValueIndex single = %d, want -1", idx)
	}
}

func TestOpenSource_NonExistent(t *testing.T) {
	_, _, err := openSource("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestOpenSource_Directory(t *testing.T) {
	dir := t.TempDir()
	src, closer, err := openSource(dir)
	if err != nil {
		t.Fatalf("openSource dir: %v", err)
	}
	if src == nil {
		t.Error("expected non-nil source")
	}
	if closer != nil {
		closer()
	}
}

func TestDefaultParseWorkers(t *testing.T) {
	cpus := runtime.NumCPU()
	if cpus < 1 {
		t.Errorf("NumCPU = %d, expected >= 1", cpus)
	}
}
