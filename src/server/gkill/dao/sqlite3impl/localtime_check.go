package sqlite3impl

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// LocaltimeAgreement は SQLite の 'localtime' 修飾子と Go の time.Local が
// 同じ瞬間を同じ壁時計時刻に写すかの検査結果。
//
// 時間帯フィルタは SQL 段（GenerateFindSQLCommon の strftime(..., 'localtime')）と
// Go 段（find_filter.go の newKyouTimeFilter が RelatedTime.In(time.Local)）の2段で判定する。
// 'localtime' は SQLite が libc の localtime_r に聞く値で、gkill の SQLite（modernc、musl 転写）は
// TZ 環境変数 → /etc/localtime → どちらも無ければ UTC で決める。Go の time.Local はそれと
// 独立に決まる（Android では main/common の fixTimezone が getprop から直す）ので、
// 両者が食い違う環境では SQL 段と Go 段が別の壁時計で判定し、9時間より狭い窓の検索が
// エラーも警告も出ないまま常に0件になる（2026-09-16 に Android で実際にそうなった）。
// documents/adr/0220-sqlite-localtime-follows-libc-zone.md
type LocaltimeAgreement struct {
	// Probe は検査に使った瞬間（固定値。実行時刻に依存させない）。
	Probe time.Time
	// GoLocal は Go の time.Local で写した "15:04:05" と曜日（0=日曜）。
	GoLocal   string
	GoWeekday int
	// SQLiteLocal は SQLite の datetime(?, 'unixepoch', 'localtime') で写した "HH:MM:SS" と strftime('%w')。
	SQLiteLocal   string
	SQLiteWeekday int
}

// Agrees は両者が一致していれば true。
func (a LocaltimeAgreement) Agrees() bool {
	return a.GoLocal == a.SQLiteLocal && a.GoWeekday == a.SQLiteWeekday
}

// localtimeProbeUnix は検査に使う固定の瞬間（2026-09-16T12:09:04+09:00 = 03:09:04Z）。
// 日付境界（UTC と +09:00 で日付が違う瞬間）を含めると曜日のずれも一緒に見える。
const localtimeProbeUnix = int64(1789528144)

// CheckLocaltimeAgreesWithGo は SQLite の 'localtime' と Go の time.Local を同じ瞬間で突き合わせる。
//
// メモリ DB を1本開いて閉じるだけで、実データには触らない。起動時に1回呼び、
// 一致しなければ gkill_error.log へ両方の値を出すのが用途（黙って0件になる代わりに、原因を1行残す）。
func CheckLocaltimeAgreesWithGo(ctx context.Context) (LocaltimeAgreement, error) {
	probe := time.Unix(localtimeProbeUnix, 0)
	goLocal := probe.In(time.Local)
	result := LocaltimeAgreement{
		Probe:     probe,
		GoLocal:   goLocal.Format("15:04:05"),
		GoWeekday: int(goLocal.Weekday()),
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return result, fmt.Errorf("error at open memory database for localtime check: %w", err)
	}
	defer db.Close()

	var sqliteWeekday string
	err = db.QueryRowContext(ctx,
		"SELECT strftime('%H:%M:%S', datetime(?, 'unixepoch', 'localtime')), strftime('%w', datetime(?, 'unixepoch', 'localtime'))",
		localtimeProbeUnix, localtimeProbeUnix,
	).Scan(&result.SQLiteLocal, &sqliteWeekday)
	if err != nil {
		return result, fmt.Errorf("error at query localtime check: %w", err)
	}
	if _, err := fmt.Sscanf(sqliteWeekday, "%d", &result.SQLiteWeekday); err != nil {
		return result, fmt.Errorf("error at parse sqlite weekday %q: %w", sqliteWeekday, err)
	}
	return result, nil
}
