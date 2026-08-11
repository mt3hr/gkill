package main

import (
	"time"

	// Windowsにはシステムのタイムゾーンデータベースが無く、
	// time.LoadLocation("Asia/Tokyo") がGoをインストールしていない環境で失敗する。
	// 埋め込んでおかないと「なぜか日付が9時間ずれる」形で静かに壊れる。
	_ "time/tzdata"
)

// parseTimestamp は Takeout のタイムスタンプを Unix 秒に変換する。
//
// 形は3種類しかない。
//   - 2024-04-03T02:05:00Z     （UTCの瞬間。per-sample のファイル）
//   - 2024-04-04T00:00:00      （オフセット無し。現地の暦日に見せかけの時刻を付けたもの）
//   - 2024-11-13               （日付のみ。同上）
//
// 24Mレコードを舐めるので time.Parse は使わない（1回あたり150〜250ns で、
// それだけで数コア秒になる）。桁を直接読んで civil days から計算する。
// 想定外の形のときだけ time.Parse にフォールバックする。
//
// dateOnly が true なら「日付だけが意味を持つ」ことを表し、
// 呼び出し側はタイムゾーン変換をしてはいけない。
func parseTimestamp(value string) (unixSec int64, dateOnly bool, ok bool) {
	if len(value) == len("2024-04-01") {
		year, month, day, parsed := parseDatePart(value)
		if !parsed {
			return 0, false, false
		}
		return daysFromCivil(year, month, day) * 86400, true, true
	}
	if len(value) < len("2024-04-01T00:00:00") {
		return fallbackParse(value)
	}
	if value[10] != 'T' {
		return fallbackParse(value)
	}
	year, month, day, parsed := parseDatePart(value[:10])
	if !parsed {
		return fallbackParse(value)
	}
	hour, ok1 := parseTwoDigits(value[11:13])
	minute, ok2 := parseTwoDigits(value[14:16])
	second, ok3 := parseTwoDigits(value[17:19])
	if !ok1 || !ok2 || !ok3 || value[13] != ':' || value[16] != ':' {
		return fallbackParse(value)
	}
	seconds := daysFromCivil(year, month, day)*86400 + int64(hour)*3600 + int64(minute)*60 + int64(second)

	// 秒の後ろに何も無ければオフセット無し = 現地の暦日
	rest := value[19:]
	switch {
	case rest == "":
		return seconds, true, true
	case rest == "Z" || rest == "z":
		return seconds, false, true
	default:
		// ミリ秒やオフセット付き。ここは件数が少ないので素直にパースする
		return fallbackParse(value)
	}
}

// fallbackParse は想定外の形を time.Parse で読む。
func fallbackParse(value string) (int64, bool, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Unix(), false, true
		}
	}
	return 0, false, false
}

func parseDatePart(value string) (year int, month int, day int, ok bool) {
	if len(value) != len("2024-04-01") || value[4] != '-' || value[7] != '-' {
		return 0, 0, 0, false
	}
	y, ok1 := parseFourDigits(value[0:4])
	m, ok2 := parseTwoDigits(value[5:7])
	d, ok3 := parseTwoDigits(value[8:10])
	if !ok1 || !ok2 || !ok3 || m < 1 || m > 12 || d < 1 || d > 31 {
		return 0, 0, 0, false
	}
	return y, m, d, true
}

func parseTwoDigits(s string) (int, bool) {
	if len(s) != 2 || s[0] < '0' || s[0] > '9' || s[1] < '0' || s[1] > '9' {
		return 0, false
	}
	return int(s[0]-'0')*10 + int(s[1]-'0'), true
}

func parseFourDigits(s string) (int, bool) {
	if len(s) != 4 {
		return 0, false
	}
	value := 0
	for i := range 4 {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		value = value*10 + int(s[i]-'0')
	}
	return value, true
}

// daysFromCivil は暦日を1970-01-01からの日数に変換する。
// Howard Hinnant の chrono アルゴリズム。
func daysFromCivil(year int, month int, day int) int64 {
	y := int64(year)
	if month <= 2 {
		y--
	}
	era := y / 400
	if y < 0 {
		era = (y - 399) / 400
	}
	yoe := y - era*400
	mp := int64((month + 9) % 12)
	doy := (153*mp+2)/5 + int64(day) - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

// localDayBucket は「Unix秒がどの現地日か」を返す。
//
// サンプルはファイル内で時系列に並んでいるので、直前に求めた1日の範囲を
// 覚えておけばほぼ毎回そのまま使える。time.Time.In(loc) が
// 24Mレコードぶん走るのを防ぐためのキャッシュ。
type localDayBucket struct {
	loc *time.Location

	curDate  string
	curStart int64
	curEnd   int64 // 排他
	hasCur   bool
}

func newLocalDayBucket(loc *time.Location) *localDayBucket {
	return &localDayBucket{loc: loc}
}

// dateOf は Unix 秒に対応する現地日付（YYYY-MM-DD）と、その日の 0時からの経過秒を返す。
func (b *localDayBucket) dateOf(unixSec int64) (string, int) {
	if b.hasCur && unixSec >= b.curStart && unixSec < b.curEnd {
		return b.curDate, int(unixSec - b.curStart)
	}
	local := time.Unix(unixSec, 0).In(b.loc)
	year, month, day := local.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, b.loc)
	// 夏時間の切り替え日は24時間ちょうどにならないので、翌日の0時から求める
	nextDay := time.Date(year, month, day+1, 0, 0, 0, 0, b.loc)

	b.curDate = startOfDay.Format("2006-01-02")
	b.curStart = startOfDay.Unix()
	b.curEnd = nextDay.Unix()
	b.hasCur = true
	return b.curDate, int(unixSec - b.curStart)
}

// noonUnixOf は現地日付の12:00の Unix 秒を返す。
//
// 0時ではなく正午にするのは、閲覧側のブラウザが自分のタイムゾーンで
// 日付を出すため。0時だと1時間西の環境で前日にずれる。
func noonUnixOf(dateLocal string, loc *time.Location) int64 {
	parsed, err := time.ParseInLocation("2006-01-02", dateLocal, loc)
	if err != nil {
		return 0
	}
	return parsed.Add(12 * time.Hour).Unix()
}

// dateOnlyPrefix は「日付だけが意味を持つ」タイムスタンプから日付部分を取り出す。
func dateOnlyPrefix(value string) string {
	if len(value) < len("2024-04-01") {
		return ""
	}
	return value[:len("2024-04-01")]
}

// loadLocation は設定のタイムゾーンを読む。空なら既定値。
func loadLocation(name string) (*time.Location, error) {
	if name == "" {
		name = defaultTimezone
	}
	return time.LoadLocation(name)
}
