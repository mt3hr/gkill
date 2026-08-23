package find

import (
	"fmt"
	"time"
)

// secondsPerDay は1日の秒数。NormalizeSecondOfDay の二重解釈の境界に使う。
const secondsPerDay = 86400

// NormalizeSecondOfDay は時間帯フィルタの秒値を「その日の何秒目か(0..86399)」へ正規化する。
//
// PeriodOfTimeStartTimeSecond / PeriodOfTimeEndTimeSecond には歴史的に2つの表現が届く:
//   - Webクライアント: 「今日のローカル h:m:s」の絶対epoch秒（moment().startOf("day").hour(h)...unix()）
//   - MCPクライアント: 0..86399 の「秒オブデイ」（MCPスキーマが最初からそう宣言していた）
//
// 0..86399 の絶対epoch秒は 1970-01-01/02 UTC にしか存在せず、時間帯フィルタとして
// その時刻を指定する実クライアントは存在しないため、86400未満は秒オブデイとして
// そのまま採用し、それ以外は従来どおり「絶対epoch秒 → ローカル時刻の時分秒」で解釈する。
// この二重解釈により、Webの既存動作を一切変えずにMCP契約のズレ（+9時間）を解消した。
// 経緯と却下案: documents/adr/0009-period-of-time-second-of-day.md
func NormalizeSecondOfDay(v int64) int {
	if v >= 0 && v < secondsPerDay {
		return int(v)
	}
	t := time.Unix(v, 0).In(time.Local)
	return t.Hour()*3600 + t.Minute()*60 + t.Second()
}

// PeriodStartSecondOfDay は時間帯フィルタの開始秒（秒オブデイ）を返す。
// 未指定なら ok=false。解釈規則は NormalizeSecondOfDay を参照。
//
// 時間帯フィルタの秒解釈はこのアクセサ（と PeriodEndSecondOfDay）だけが正本。
// 以前は find_filter.go / sqlite3impl_util.go / git_commit_log_repository_local_dir_impl.go の
// 3箇所に同じ変換が写経されており、契約の食い違いに気付けなかった。
func (q *FindQuery) PeriodStartSecondOfDay() (sec int, ok bool) {
	if q.PeriodOfTimeStartTimeSecond == nil {
		return 0, false
	}
	return NormalizeSecondOfDay(*q.PeriodOfTimeStartTimeSecond), true
}

// PeriodEndSecondOfDay は時間帯フィルタの終了秒（秒オブデイ）を返す。
// 未指定なら ok=false。解釈規則は NormalizeSecondOfDay を参照。
func (q *FindQuery) PeriodEndSecondOfDay() (sec int, ok bool) {
	if q.PeriodOfTimeEndTimeSecond == nil {
		return 0, false
	}
	return NormalizeSecondOfDay(*q.PeriodOfTimeEndTimeSecond), true
}

// SecondOfDayToHHMMSS は秒オブデイを "15:04:05" 形式の文字列にする。
// SQL経路の strftime('%H:%M:%S', ...) との文字列比較にそのまま使える。
func SecondOfDayToHHMMSS(sec int) string {
	return fmt.Sprintf("%02d:%02d:%02d", sec/3600, (sec/60)%60, sec%60)
}
