package find

import (
	"testing"
	"time"
)

// NormalizeSecondOfDay の二重解釈の境界を固定する。
//
//	0..86399   = 秒オブデイそのまま（MCP契約）
//	86400以上  = 絶対epoch秒 → ローカル時刻の時分秒（Web契約）
//	負値       = 絶対epoch秒として扱う（1970以前。実クライアントは送らないが未定義にしない）
//
// この境界を動かすと、MCPの時間帯検索が9時間ずれるか、Webの時間帯検索が9時間ずれるかの
// どちらかが静かに起きる。経緯: documents/adr/0108-period-of-time-second-of-day.md
func TestNormalizeSecondOfDay(t *testing.T) {
	// epoch解釈の期待値はTZ依存なので、テスト自身がローカルTZで計算する
	epochOf := func(hour, minute, second int) int64 {
		return time.Date(2026, 8, 23, hour, minute, second, 0, time.Local).Unix()
	}

	for _, c := range []struct {
		name string
		in   int64
		want int
	}{
		{name: "0は秒オブデイの0（epoch 1970-01-01T00:00:00Zではない）", in: 0, want: 0},
		{name: "09:00:00の秒オブデイ", in: 32400, want: 32400},
		{name: "上端86399は秒オブデイ", in: 86399, want: 86399},
		{name: "86400はepoch解釈に切り替わる", in: 86400, want: NormalizeSecondOfDay(86400)},
		{name: "今日のローカル09:30:15のepoch（Web契約）", in: epochOf(9, 30, 15), want: 9*3600 + 30*60 + 15},
		{name: "今日のローカル00:00:00のepoch（Web契約）", in: epochOf(0, 0, 0), want: 0},
		{name: "今日のローカル23:59:59のepoch（Web契約）", in: epochOf(23, 59, 59), want: 23*3600 + 59*60 + 59},
		{name: "負値はepoch解釈", in: -1, want: func() int {
			lt := time.Unix(-1, 0).In(time.Local)
			return lt.Hour()*3600 + lt.Minute()*60 + lt.Second()
		}()},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := NormalizeSecondOfDay(c.in); got != c.want {
				t.Errorf("NormalizeSecondOfDay(%d) = %d, want %d", c.in, got, c.want)
			}
		})
	}

	// 86400のepoch解釈が「ローカルにおける 1970-01-02T00:00:00Z の時分秒」であることを
	// 参照実装で確かめる（JSTなら 09:00:00 = 32400）
	lt := time.Unix(86400, 0).In(time.Local)
	want := lt.Hour()*3600 + lt.Minute()*60 + lt.Second()
	if got := NormalizeSecondOfDay(86400); got != want {
		t.Errorf("NormalizeSecondOfDay(86400) = %d, want %d (epoch解釈)", got, want)
	}
}

// アクセサは nil のとき ok=false、非nilのとき正規化済みの値を返す。
func TestPeriodSecondOfDayAccessors(t *testing.T) {
	q := &FindQuery{}
	if _, ok := q.PeriodStartSecondOfDay(); ok {
		t.Error("nil の PeriodStartSecondOfDay が ok=true を返した")
	}
	if _, ok := q.PeriodEndSecondOfDay(); ok {
		t.Error("nil の PeriodEndSecondOfDay が ok=true を返した")
	}

	startSecOfDay := int64(32400) // 09:00:00（秒オブデイ）
	endEpoch := time.Date(2026, 8, 23, 10, 0, 0, 0, time.Local).Unix()
	q = &FindQuery{
		PeriodOfTimeStartTimeSecond: &startSecOfDay,
		PeriodOfTimeEndTimeSecond:   &endEpoch,
	}
	if sec, ok := q.PeriodStartSecondOfDay(); !ok || sec != 32400 {
		t.Errorf("PeriodStartSecondOfDay = (%d, %v), want (32400, true)", sec, ok)
	}
	if sec, ok := q.PeriodEndSecondOfDay(); !ok || sec != 10*3600 {
		t.Errorf("PeriodEndSecondOfDay = (%d, %v), want (36000, true)", sec, ok)
	}
}

func TestSecondOfDayToHHMMSS(t *testing.T) {
	for _, c := range []struct {
		in   int
		want string
	}{
		{in: 0, want: "00:00:00"},
		{in: 32400, want: "09:00:00"},
		{in: 86399, want: "23:59:59"},
		{in: 3661, want: "01:01:01"},
	} {
		if got := SecondOfDayToHHMMSS(c.in); got != c.want {
			t.Errorf("SecondOfDayToHHMMSS(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

// 正準rep種別のリストと判定関数の整合。
// selectMatchRepsFromQuery の switch との集合一致は api パッケージ側の
// TestKyouRepTypesCoversRepsOfKyouRepType が固定する。
func TestIsKyouRepType(t *testing.T) {
	for _, repType := range KyouRepTypes {
		if !IsKyouRepType(repType) {
			t.Errorf("IsKyouRepType(%q) = false, want true", repType)
		}
	}
	for _, unknown := range []string{"", "Kmemo", "idf", "plugin", "tag", "text", "notification", "gpslog"} {
		if IsKyouRepType(unknown) {
			t.Errorf("IsKyouRepType(%q) = true, want false", unknown)
		}
	}
}
