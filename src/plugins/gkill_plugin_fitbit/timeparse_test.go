package main

import (
	"testing"
	"time"
)

func TestParseTimestamp(t *testing.T) {
	cases := []struct {
		value        string
		wantUnix     int64
		wantDateOnly bool
		wantOK       bool
	}{
		// UTCの瞬間
		{"2024-04-03T02:05:00Z", time.Date(2024, 4, 3, 2, 5, 0, 0, time.UTC).Unix(), false, true},
		// オフセット無し = 現地の暦日に見せかけの時刻を付けたもの
		{"2024-04-04T00:00:00", time.Date(2024, 4, 4, 0, 0, 0, 0, time.UTC).Unix(), true, true},
		// 日付のみ
		{"2024-11-13", time.Date(2024, 11, 13, 0, 0, 0, 0, time.UTC).Unix(), true, true},
		// 壊れているもの
		{"", 0, false, false},
		{"not-a-time", 0, false, false},
	}
	for _, c := range cases {
		unix, dateOnly, ok := parseTimestamp(c.value)
		if ok != c.wantOK {
			t.Errorf("parseTimestamp(%q) ok = %v, want %v", c.value, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if unix != c.wantUnix || dateOnly != c.wantDateOnly {
			t.Errorf("parseTimestamp(%q) = %d, %v; want %d, %v", c.value, unix, dateOnly, c.wantUnix, c.wantDateOnly)
		}
	}
}

// TestParseTimestamp_MatchesTimeParse は、速い経路と time.Parse が
// 同じ結果になることを確認する。
// 24Mレコードのために手で桁を読んでいるので、ここがずれると全部ずれる。
func TestParseTimestamp_MatchesTimeParse(t *testing.T) {
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 5000 {
		moment := base.Add(time.Duration(i) * 97 * time.Minute)
		value := moment.Format("2006-01-02T15:04:05Z")
		unix, dateOnly, ok := parseTimestamp(value)
		if !ok {
			t.Fatalf("parseTimestamp(%q) が失敗した", value)
		}
		if dateOnly {
			t.Fatalf("parseTimestamp(%q) が dateOnly を返した", value)
		}
		if unix != moment.Unix() {
			t.Fatalf("parseTimestamp(%q) = %d, want %d", value, unix, moment.Unix())
		}
	}
}

// TestParseTimestamp_WithFraction はミリ秒付きがフォールバックで読めることを確認する。
func TestParseTimestamp_WithFraction(t *testing.T) {
	unix, _, ok := parseTimestamp("2026-06-20T15:23:10.829Z")
	if !ok {
		t.Fatal("ミリ秒付きが読めない")
	}
	want := time.Date(2026, 6, 20, 15, 23, 10, 0, time.UTC).Unix()
	if unix != want {
		t.Errorf("= %d, want %d", unix, want)
	}
}

// TestLocalDayBucket_JSTBoundary は、UTCの15:00がJSTの翌日になることを確認する。
// ここを間違えると、日次の集計が丸ごと1日ずれる。
func TestLocalDayBucket_JSTBoundary(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		// time/tzdata を埋め込んであるので、ここで失敗したら埋め込みが外れている
		t.Fatalf("Asia/Tokyo を読めない（time/tzdata の埋め込みが外れている）: %v", err)
	}
	bucket := newLocalDayBucket(loc)

	date, secondsOfDay := bucket.dateOf(time.Date(2024, 4, 3, 14, 59, 0, 0, time.UTC).Unix())
	if date != "2024-04-03" {
		t.Errorf("14:59Z の現地日 = %q, want 2024-04-03", date)
	}
	if secondsOfDay/3600 != 23 {
		t.Errorf("14:59Z の現地時 = %d時, want 23時", secondsOfDay/3600)
	}

	date, secondsOfDay = bucket.dateOf(time.Date(2024, 4, 3, 15, 0, 0, 0, time.UTC).Unix())
	if date != "2024-04-04" {
		t.Errorf("15:00Z の現地日 = %q, want 2024-04-04", date)
	}
	if secondsOfDay != 0 {
		t.Errorf("15:00Z の現地0時からの経過 = %d秒, want 0", secondsOfDay)
	}
}

// TestLocalDayBucket_DST は、夏時間の切り替え日でも日付が壊れないことを確認する。
// 1日が24時間ちょうどではないので、単純な加算で境界を求めるとずれる。
func TestLocalDayBucket_DST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("America/New_York を読めない: %v", err)
	}
	bucket := newLocalDayBucket(loc)

	// 2024-03-10 は米東部の夏時間開始日（23時間の日）
	for hour := range 30 {
		moment := time.Date(2024, 3, 10, 0, 0, 0, 0, loc).Add(time.Duration(hour) * time.Hour)
		date, secondsOfDay := bucket.dateOf(moment.Unix())
		wantDate := moment.In(loc).Format("2006-01-02")
		if date != wantDate {
			t.Errorf("%v の現地日 = %q, want %q", moment, date, wantDate)
		}
		if secondsOfDay < 0 {
			t.Errorf("%v の経過秒が負: %d", moment, secondsOfDay)
		}
	}
}

// TestNoonUnixOf は関連時刻が正午になることを確認する。
// 0時にすると、閲覧側が1時間西のタイムゾーンだと前日にずれる。
func TestNoonUnixOf(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("Asia/Tokyo を読めない: %v", err)
	}
	unix := noonUnixOf("2026-08-09", loc)
	got := time.Unix(unix, 0).In(loc)
	if got.Hour() != 12 || got.Format("2006-01-02") != "2026-08-09" {
		t.Errorf("noonUnixOf = %v, want 2026-08-09 12:00 JST", got)
	}
}

func TestDaysFromCivil(t *testing.T) {
	cases := []struct {
		year, month, day int
		want             int64
	}{
		{1970, 1, 1, 0},
		{1970, 1, 2, 1},
		{2000, 1, 1, 10957},
		{2024, 4, 3, 19816},
	}
	for _, c := range cases {
		if got := daysFromCivil(c.year, c.month, c.day); got != c.want {
			t.Errorf("daysFromCivil(%d,%d,%d) = %d, want %d", c.year, c.month, c.day, got, c.want)
		}
	}
}
