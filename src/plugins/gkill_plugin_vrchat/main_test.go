package main

import (
	"os"
	"testing"
	"time"
)

// TestMain はログの時刻を UTC で読むようにしてから走らせる。
// ログの時刻はタイムゾーンを持たないので、テストの期待値を PC のゾーンに依らせないため。
func TestMain(m *testing.M) {
	logLocationOverride = time.UTC
	os.Exit(m.Run())
}

// testTime はテストの固定時刻（UTC）。テストデータの時刻と同じ書き方で読む。
func testTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation(logTimeLayout, value, time.UTC)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return parsed
}
