package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// normalFixture は正常終了する起動のログ（合成値）。
func normalFixture() string {
	return filepath.Join("testdata", "fixtures", "output_log_2024-01-01_10-00-00.txt")
}

// crashFixture は終了の行が無い（落ちた）起動のログ（合成値）。
func crashFixture() string {
	return filepath.Join("testdata", "fixtures", "output_log_2024-01-02_10-00-00.txt")
}

func countEvents(parsed parsedLog, kind int) int {
	count := 0
	for _, event := range parsed.Events {
		if event.Kind == kind {
			count++
		}
	}
	return count
}

// 拾う行だけを出来事にし、時刻の無い継続行やネットワークの状態の Switching は拾わない。
func TestParseLogPicksKnownLines(t *testing.T) {
	parsed, err := parseLogFile(normalFixture())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.SessionKey != "2024.01.01 10:00:00" {
		t.Errorf("SessionKey = %q, want 最初の時刻付きの行の時刻", parsed.SessionKey)
	}
	if _, ok := parsed.SelfNames["testuser"]; !ok {
		t.Errorf("SelfNames = %v, want testuser", parsed.SelfNames)
	}
	if !parsed.LastTime.Equal(testTime(t, "2024.01.01 11:05:00")) {
		t.Errorf("LastTime = %v", parsed.LastTime)
	}
	cases := []struct {
		kind int
		want int
		name string
	}{
		{evJoining, 2, "Joining"},
		{evEnteringRoom, 2, "Entering Room"},
		{evLeftRoom, 1, "OnLeftRoom"},
		{evPlayerJoined, 7, "OnPlayerJoined"},
		{evPlayerLeft, 3, "OnPlayerLeft"},
		{evAvatar, 2, "アバターの切り替え（自分以外も拾う。絞るのは記録を作るとき）"},
		{evVideo, 2, "動画"},
		{evQuit, 1, "終了"},
	}
	for _, c := range cases {
		if got := countEvents(parsed, c.kind); got != c.want {
			t.Errorf("%s = %d件, want %d件", c.name, got, c.want)
		}
	}
	for _, event := range parsed.Events {
		if event.Kind == evJoining && event.ID == "wrld_00000000-0000-4000-8000-000000000001" && event.Value != "10000~region(jp)" {
			t.Errorf("インスタンス = %q", event.Value)
		}
	}
}

// 長すぎる行は読み捨てて数え、その次の行からは普通に読む。CRLF の行も読める。
func TestParseLogDropsTooLongLines(t *testing.T) {
	long := "2024.01.01 10:00:00 Debug      -  " + strings.Repeat("x", maxLineBytes+10)
	input := long + "\r\n" +
		"2024.01.01 10:01:00 Debug      -  [Behaviour] Entering Room: WorldAlpha\r\n"
	parsed, err := parseLog(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Dropped != 1 {
		t.Errorf("Dropped = %d, want 1", parsed.Dropped)
	}
	if countEvents(parsed, evEnteringRoom) != 1 || parsed.Events[0].Name != "WorldAlpha" {
		t.Errorf("長い行のあとの行が読めていない: %+v", parsed.Events)
	}
	if parsed.SessionKey != "2024.01.01 10:01:00" {
		t.Errorf("SessionKey = %q, want 読み捨てた行ではなく次の行の時刻", parsed.SessionKey)
	}
}

// 時刻付きの行が1つも無いファイルは起動キーが空で、記録を作らない。
func TestParseLogWithoutTimestamp(t *testing.T) {
	parsed, err := parseLog(strings.NewReader("not a log line\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.SessionKey != "" || len(parsed.Events) != 0 {
		t.Errorf("parsed = %+v, want 空", parsed)
	}
	if got := buildRecords(parsed); len(got) != 0 {
		t.Errorf("buildRecords = %d件, want 0件", len(got))
	}
}
