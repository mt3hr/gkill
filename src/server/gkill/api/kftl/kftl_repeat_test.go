package kftl

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// 繰り返しブロック「？？」の純ロジック（条件・回数/終了日・起点・候補日時）を固定する。
// クライアント側 src/client/__tests__/unit/kftl/kftl-repeat.test.ts と対のテーブル。

func TestParseRepeatCondition(t *testing.T) {
	cases := []struct {
		in       string
		kind     repeatCondKind
		weekdays []time.Weekday
		interval int
		nth      int
		monthDay int
	}{
		{in: "毎日", kind: repeatCondDaily},
		{in: "daily", kind: repeatCondDaily},
		{in: "金", kind: repeatCondWeekday, weekdays: []time.Weekday{time.Friday}, interval: 1},
		{in: "fri", kind: repeatCondWeekday, weekdays: []time.Weekday{time.Friday}, interval: 1},
		{in: "月水金", kind: repeatCondWeekday, weekdays: []time.Weekday{time.Monday, time.Wednesday, time.Friday}, interval: 1},
		{in: "mon,wed,fri", kind: repeatCondWeekday, weekdays: []time.Weekday{time.Monday, time.Wednesday, time.Friday}, interval: 1},
		{in: "2週月", kind: repeatCondWeekday, weekdays: []time.Weekday{time.Monday}, interval: 2},
		{in: "6週金", kind: repeatCondWeekday, weekdays: []time.Weekday{time.Friday}, interval: 6},
		{in: "6w fri", kind: repeatCondWeekday, weekdays: []time.Weekday{time.Friday}, interval: 6},
		{in: "毎月15", kind: repeatCondMonthDay, monthDay: 15},
		{in: "monthly 15", kind: repeatCondMonthDay, monthDay: 15},
		{in: "第2金", kind: repeatCondNthWeekday, weekdays: []time.Weekday{time.Friday}, nth: 2},
		{in: "2nd fri", kind: repeatCondNthWeekday, weekdays: []time.Weekday{time.Friday}, nth: 2},
		{in: "最終金", kind: repeatCondNthWeekday, weekdays: []time.Weekday{time.Friday}, nth: -1},
		{in: "last fri", kind: repeatCondNthWeekday, weekdays: []time.Weekday{time.Friday}, nth: -1},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := parseRepeatCondition(c.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.kind != c.kind {
				t.Errorf("kind = %v, want %v", got.kind, c.kind)
			}
			if len(got.weekdays) != len(c.weekdays) {
				t.Fatalf("weekdays = %v, want %v", got.weekdays, c.weekdays)
			}
			for i := range c.weekdays {
				if got.weekdays[i] != c.weekdays[i] {
					t.Errorf("weekdays[%d] = %v, want %v", i, got.weekdays[i], c.weekdays[i])
				}
			}
			if got.weekInterval != c.interval {
				t.Errorf("weekInterval = %d, want %d", got.weekInterval, c.interval)
			}
			if got.nth != c.nth {
				t.Errorf("nth = %d, want %d", got.nth, c.nth)
			}
			if got.monthDay != c.monthDay {
				t.Errorf("monthDay = %d, want %d", got.monthDay, c.monthDay)
			}
		})
	}
}

func TestParseRepeatCondition_Invalid(t *testing.T) {
	// 「N週」は基準の週を1つに決めないといけないので、曜日を複数書けない
	for _, in := range []string{"", "   ", "きのう", "2週月水", "毎月0", "毎月32", "第6金", "6th fri", "monday", "毎月"} {
		t.Run(in, func(t *testing.T) {
			if _, err := parseRepeatCondition(in); err == nil {
				t.Errorf("%q: エラーになること", in)
			}
		})
	}
}

func TestParseRepeatCountOrUntil(t *testing.T) {
	base := time.Date(2026, 9, 2, 10, 0, 0, 0, time.Local)

	t.Run("整数は回数", func(t *testing.T) {
		n, until, err := parseRepeatCountOrUntil("3", base)
		if err != nil || n != 3 || until != nil {
			t.Fatalf("n=%d until=%v err=%v", n, until, err)
		}
	})
	// 日付のみの終了日はその日の終わりまで伸ばす。伸ばさないと当日18:00が範囲外になる
	t.Run("日付のみの終了日は23:59:59まで伸びる", func(t *testing.T) {
		n, until, err := parseRepeatCountOrUntil("2026-12-31", base)
		if err != nil || n != 0 || until == nil {
			t.Fatalf("n=%d until=%v err=%v", n, until, err)
		}
		want := time.Date(2026, 12, 31, 23, 59, 59, 0, time.Local)
		if !until.Equal(want) {
			t.Errorf("until = %v, want %v", until, want)
		}
	})
	t.Run("時刻つきの終了日はそのまま", func(t *testing.T) {
		_, until, err := parseRepeatCountOrUntil("2026-12-31 12:00", base)
		if err != nil || until == nil {
			t.Fatalf("until=%v err=%v", until, err)
		}
		want := time.Date(2026, 12, 31, 12, 0, 0, 0, time.Local)
		if !until.Equal(want) {
			t.Errorf("until = %v, want %v", until, want)
		}
	})
	t.Run("範囲外の回数と読めない行はエラー", func(t *testing.T) {
		for _, in := range []string{"", "0", "-1", "1001", "みっつ"} {
			if _, _, err := parseRepeatCountOrUntil(in, base); err == nil {
				t.Errorf("%q: エラーになること", in)
			}
		}
	})
}

func TestParseRepeatAddIfExists(t *testing.T) {
	for _, in := range []string{"", "  ", "no", "NO", "いいえ"} {
		got, err := parseRepeatAddIfExists(in)
		if err != nil || got {
			t.Errorf("%q: 既定は追加しない (got=%v err=%v)", in, got, err)
		}
	}
	for _, in := range []string{"yes", "YES", "はい"} {
		got, err := parseRepeatAddIfExists(in)
		if err != nil || !got {
			t.Errorf("%q: 追加する (got=%v err=%v)", in, got, err)
		}
	}
	if _, err := parseRepeatAddIfExists("maybe"); err == nil {
		t.Error("yes/no 以外はエラーになること")
	}
}

func TestParseRepeatOrigin(t *testing.T) {
	base := time.Date(2026, 9, 2, 10, 0, 0, 0, time.Local)
	if got, err := parseRepeatOrigin("", base); err != nil || got != nil {
		t.Errorf("空は nil (got=%v err=%v)", got, err)
	}
	got, err := parseRepeatOrigin("2026-09-12 18:00", base)
	if err != nil || got == nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
	want := time.Date(2026, 9, 12, 18, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("origin = %v, want %v", got, want)
	}
	if _, err := parseRepeatOrigin("きのう", base); err == nil {
		t.Error("読めない起点はエラーになること")
	}
}

// ─── 候補日時 ─────────────────────────────────────────────────────────────────

func ymdhm(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, time.Local)
}

func helperOccurrences(t *testing.T, condText string, count int, anchor, origin time.Time) []time.Time {
	t.Helper()
	cond, err := parseRepeatCondition(condText)
	if err != nil {
		t.Fatalf("parseRepeatCondition(%q): %v", condText, err)
	}
	spec := &repeatSpec{cond: cond, count: count, origin: &origin}
	got, err := occurrencesOf(spec, anchor, origin)
	if err != nil {
		t.Fatalf("occurrencesOf: %v", err)
	}
	return got
}

func assertTimes(t *testing.T, got []time.Time, want []time.Time) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("件数 = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// 2026-09-02 は水曜。時刻はアンカーから取り、起点ちょうどは含めない。
func TestOccurrencesOf(t *testing.T) {
	origin := ymdhm(2026, 9, 2, 10, 0)

	t.Run("毎日。アンカーが起点より後なら当日も入る", func(t *testing.T) {
		got := helperOccurrences(t, "毎日", 3, ymdhm(2026, 9, 2, 18, 0), origin)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 9, 2, 18, 0), ymdhm(2026, 9, 3, 18, 0), ymdhm(2026, 9, 4, 18, 0),
		})
	})

	t.Run("毎日。アンカーが起点より前なら当日は落ちる", func(t *testing.T) {
		got := helperOccurrences(t, "毎日", 3, ymdhm(2026, 9, 2, 8, 0), origin)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 9, 3, 8, 0), ymdhm(2026, 9, 4, 8, 0), ymdhm(2026, 9, 5, 8, 0),
		})
	})

	t.Run("曜日", func(t *testing.T) {
		got := helperOccurrences(t, "金", 3, ymdhm(2026, 9, 2, 18, 0), origin)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 9, 11, 18, 0), ymdhm(2026, 9, 18, 18, 0),
		})
	})

	t.Run("複数曜日は書いた順ではなく日付順", func(t *testing.T) {
		got := helperOccurrences(t, "月水金", 4, ymdhm(2026, 9, 2, 18, 0), origin)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 9, 2, 18, 0), ymdhm(2026, 9, 4, 18, 0),
			ymdhm(2026, 9, 7, 18, 0), ymdhm(2026, 9, 9, 18, 0),
		})
	})

	t.Run("N週おきは最初の一致を基準にする", func(t *testing.T) {
		got := helperOccurrences(t, "6週金", 3, ymdhm(2026, 9, 2, 18, 0), origin)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 10, 16, 18, 0), ymdhm(2026, 11, 27, 18, 0),
		})
	})

	t.Run("毎月N日", func(t *testing.T) {
		got := helperOccurrences(t, "毎月15", 3, ymdhm(2026, 9, 2, 18, 0), origin)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 9, 15, 18, 0), ymdhm(2026, 10, 15, 18, 0), ymdhm(2026, 11, 15, 18, 0),
		})
	})

	t.Run("第N曜日", func(t *testing.T) {
		got := helperOccurrences(t, "第2金", 3, ymdhm(2026, 9, 2, 18, 0), origin)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 9, 11, 18, 0), ymdhm(2026, 10, 9, 18, 0), ymdhm(2026, 11, 13, 18, 0),
		})
	})

	t.Run("最終曜日", func(t *testing.T) {
		got := helperOccurrences(t, "最終金", 3, ymdhm(2026, 9, 2, 18, 0), origin)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 9, 25, 18, 0), ymdhm(2026, 10, 30, 18, 0), ymdhm(2026, 11, 27, 18, 0),
		})
	})

	// 存在しない日はその月を飛ばす。最も近い日へ丸めると第4金・毎月30と重複する
	t.Run("毎月31は31日の無い月を飛ばす", func(t *testing.T) {
		jan := ymdhm(2026, 1, 1, 0, 0)
		got := helperOccurrences(t, "毎月31", 3, ymdhm(2026, 1, 1, 9, 0), jan)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 1, 31, 9, 0), ymdhm(2026, 3, 31, 9, 0), ymdhm(2026, 5, 31, 9, 0),
		})
	})

	t.Run("第5金は第5金の無い月を飛ばす", func(t *testing.T) {
		jan := ymdhm(2026, 1, 1, 0, 0)
		got := helperOccurrences(t, "第5金", 3, ymdhm(2026, 1, 1, 9, 0), jan)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 1, 30, 9, 0), ymdhm(2026, 5, 29, 9, 0), ymdhm(2026, 7, 31, 9, 0),
		})
	})

	t.Run("起点ちょうどは含めない", func(t *testing.T) {
		got := helperOccurrences(t, "毎日", 2, ymdhm(2026, 9, 2, 10, 0), origin)
		assertTimes(t, got, []time.Time{
			ymdhm(2026, 9, 3, 10, 0), ymdhm(2026, 9, 4, 10, 0),
		})
	})
}

func TestOccurrencesOf_Until(t *testing.T) {
	origin := ymdhm(2026, 9, 2, 10, 0)
	cond, err := parseRepeatCondition("金")
	if err != nil {
		t.Fatal(err)
	}
	until := time.Date(2026, 9, 15, 23, 59, 59, 0, time.Local)
	spec := &repeatSpec{cond: cond, until: &until, origin: &origin}
	got, err := occurrencesOf(spec, ymdhm(2026, 9, 2, 18, 0), origin)
	if err != nil {
		t.Fatal(err)
	}
	assertTimes(t, got, []time.Time{ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 9, 11, 18, 0)})
}

// 上限は黙って切り詰めず入力エラーにする（部分確定するので大量生成の途中失敗が痛い）。
func TestOccurrencesOf_LimitExceeded(t *testing.T) {
	origin := ymdhm(2026, 9, 2, 10, 0)
	cond, err := parseRepeatCondition("毎日")
	if err != nil {
		t.Fatal(err)
	}
	until := time.Date(2036, 9, 2, 23, 59, 59, 0, time.Local) // 約10年ぶん = 3600件超
	spec := &repeatSpec{cond: cond, until: &until, origin: &origin}
	if _, err := occurrencesOf(spec, ymdhm(2026, 9, 2, 18, 0), origin); err == nil {
		t.Fatal("上限超過がエラーにならなかった")
	} else {
		inputErrors := CollectKFTLInputErrors(err)
		if len(inputErrors) != 1 || inputErrors[0].MessageID != "KFTL_REPEAT_LIMIT_EXCEEDED_MESSAGE_TITLE" {
			t.Errorf("MessageID = %v", inputErrors)
		}
	}
}

// ─── 行の並びと展開（結合）─────────────────────────────────────────────────────

// 2026-09-02 10:00（水）を基準にする。helperGenerateLines の base は UTC なので使わない。
func repeatTestBase() time.Time { return time.Date(2026, 9, 2, 10, 0, 0, 0, time.Local) }

func helperGenerateLinesAt(t *testing.T, text string, base time.Time) []KFTLStatementLine {
	t.Helper()
	stmt := &KFTLStatement{StatementText: text}
	factory := newKFTLFactory()
	factory.reset()
	lines, err := stmt.generateKFTLLines(
		factory, sqlite3impl.GenerateNewID(), base,
		&reps.GkillRepositories{}, &user_config.ApplicationConfig{},
		"test-user", "test-device", "test-app", "ja",
	)
	if err != nil {
		t.Fatalf("generateKFTLLines error: %v", err)
	}
	return lines
}

// helperExpand は行の解釈から繰り返しの展開まで通し、展開後のリクエスト列を返す。
func helperExpand(t *testing.T, text string) ([]KFTLRequest, error) {
	t.Helper()
	base := repeatTestBase()
	lines := helperGenerateLinesAt(t, text, base)
	requestMap := NewKFTLRequestMap()
	for _, line := range lines {
		if err := line.ApplyThisLineToRequestMap(context.Background(), requestMap); err != nil {
			return nil, err
		}
	}
	if err := expandRepeats(context.Background(), requestMap, base); err != nil {
		return nil, err
	}
	return requestMap.All(), nil
}

func helperExpandOK(t *testing.T, text string) []KFTLRequest {
	t.Helper()
	got, err := helperExpand(t, text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return got
}

func assertInputErrorID(t *testing.T, err error, wantID string) {
	t.Helper()
	if err == nil {
		t.Fatalf("エラーになること (want %s)", wantID)
	}
	inputErrors := CollectKFTLInputErrors(err)
	if len(inputErrors) == 0 {
		t.Fatalf("入力エラーであること: %v", err)
	}
	if inputErrors[0].MessageID != wantID {
		t.Errorf("MessageID = %q, want %q", inputErrors[0].MessageID, wantID)
	}
}

func TestStatement_RepeatBlockLineLabels(t *testing.T) {
	text := "ーみ\n週報\n仕事\n18:00\n？？\n金\n3\nno\n2026-09-12\n？？"
	lines := helperGenerateLinesAt(t, text, repeatTestBase())
	want := []string{
		"mi", "miTitle", "miBoardName", "miEstimateStartTime",
		"repeat", "repeatCondition", "repeatCount", "repeatAddIfExists", "repeatOrigin", "endRepeat",
	}
	if len(lines) != len(want) {
		t.Fatalf("行数 = %d, want %d", len(lines), len(want))
	}
	for i, w := range want {
		if lines[i].GetLabelName() != w {
			t.Errorf("行%d = %s, want %s", i, lines[i].GetLabelName(), w)
		}
	}
}

// 閉じる行を最優先で見るので、3行目・4行目を省いて早く閉じられる。
func TestStatement_RepeatBlockCanCloseEarly(t *testing.T) {
	text := "ーみ\n週報\n仕事\n18:00\n？？\n金\n3\n？？"
	lines := helperGenerateLinesAt(t, text, repeatTestBase())
	if got := lines[len(lines)-1].GetLabelName(); got != "endRepeat" {
		t.Errorf("最後の行 = %s, want endRepeat", got)
	}
}

func helperMiRequests(reqs []KFTLRequest) []*kftlMiRequest {
	var out []*kftlMiRequest
	for _, r := range reqs {
		if mi, ok := r.(*kftlMiRequest); ok {
			out = append(out, mi)
		}
	}
	return out
}

// 全日時欄を同じ日数だけずらす。欄どうしの相対差（開始→終了→期限）は保たれる。
func TestExpand_MiRepeatShiftsAllDateFieldsTogether(t *testing.T) {
	text := "ーみ\n週報\n仕事\n18:00\n19:00\n20:00\n？？\n金\n2\n？？"
	mis := helperMiRequests(helperExpandOK(t, text))
	if len(mis) != 2 {
		t.Fatalf("件数 = %d, want 2", len(mis))
	}
	wants := []struct{ start, end, limit time.Time }{
		{ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 9, 4, 19, 0), ymdhm(2026, 9, 4, 20, 0)},
		{ymdhm(2026, 9, 11, 18, 0), ymdhm(2026, 9, 11, 19, 0), ymdhm(2026, 9, 11, 20, 0)},
	}
	for i, w := range wants {
		if mis[i].estimateStartTime == nil || !mis[i].estimateStartTime.Equal(w.start) {
			t.Errorf("[%d] 見積開始 = %v, want %v", i, mis[i].estimateStartTime, w.start)
		}
		if mis[i].estimateEndTime == nil || !mis[i].estimateEndTime.Equal(w.end) {
			t.Errorf("[%d] 見積終了 = %v, want %v", i, mis[i].estimateEndTime, w.end)
		}
		if mis[i].limitTime == nil || !mis[i].limitTime.Equal(w.limit) {
			t.Errorf("[%d] 期限 = %v, want %v", i, mis[i].limitTime, w.limit)
		}
	}
	// タイトル・板名は全回に引き継がれる。IDは回ごとに別
	if mis[0].RequestID == mis[1].RequestID {
		t.Error("複製が同じIDを持っている")
	}
	for i := range mis {
		if mis[i].title != "週報" || mis[i].boardName != "仕事" {
			t.Errorf("[%d] title=%q board=%q", i, mis[i].title, mis[i].boardName)
		}
	}
}

// タグ・テキストと同じく項目の位置を消費しない。閉じたあとは同じ項目位置へ戻る。
func TestExpand_RepeatDoesNotConsumeMiFieldPosition(t *testing.T) {
	text := "ーみ\n週報\n仕事\n？？\n金\n2\n？？\n18:00"
	mis := helperMiRequests(helperExpandOK(t, text))
	if len(mis) != 2 {
		t.Fatalf("件数 = %d, want 2", len(mis))
	}
	if mis[0].estimateStartTime == nil || !mis[0].estimateStartTime.Equal(ymdhm(2026, 9, 4, 18, 0)) {
		t.Errorf("見積開始 = %v, want 2026-09-04 18:00（ブロックへ戻って項目行として読まれること）", mis[0].estimateStartTime)
	}
}

// 支出ブロックは全支払いが1グループ。店名・関連時刻と同じブロック共有の扱い。
func TestExpand_NlogRepeatCoversWholeBlock(t *testing.T) {
	text := "ーん\nスーパー\n牛乳\n200\nパン\n300\n？？\n金\n3\n？？"
	var nlogs []*kftlNlogRequest
	for _, r := range helperExpandOK(t, text) {
		if n, ok := r.(*kftlNlogRequest); ok {
			nlogs = append(nlogs, n)
		}
	}
	if len(nlogs) != 6 {
		t.Fatalf("件数 = %d, want 6（支払い2件 × 3回）", len(nlogs))
	}
	wantDays := []time.Time{
		ymdhm(2026, 9, 4, 10, 0), ymdhm(2026, 9, 4, 10, 0),
		ymdhm(2026, 9, 11, 10, 0), ymdhm(2026, 9, 11, 10, 0),
		ymdhm(2026, 9, 18, 10, 0), ymdhm(2026, 9, 18, 10, 0),
	}
	wantTitles := []string{"牛乳", "パン", "牛乳", "パン", "牛乳", "パン"}
	for i := range nlogs {
		if got := nlogs[i].GetRelatedTime(); !got.Equal(wantDays[i]) {
			t.Errorf("[%d] 関連時刻 = %v, want %v", i, got, wantDays[i])
		}
		if nlogs[i].title != wantTitles[i] {
			t.Errorf("[%d] 品名 = %q, want %q", i, nlogs[i].title, wantTitles[i])
		}
		if nlogs[i].block.shop != "スーパー" {
			t.Errorf("[%d] 店名 = %q", i, nlogs[i].block.shop)
		}
	}
}

// 打刻は開始時刻を基準にし、開始と終了を同じ日数だけずらす。
// TS 側は start_time が do_request まで空で、それをアンカーにして 1970 からの日数ぶんずれ
// 2026-09-10 に送った打刻が 2083 年で登録された。Go は開始時刻行が startTime にも書くので
// 元から正しいが、同じケース表（利用者の入力そのもの）で年が変わらないことを両側で固定する。
// TS 側の対: kftl-repeat-statement.test.ts「打刻は開始時刻を基準にし、開始と終了を同じ日数だけずらす」
func TestExpand_TimeIsRepeatShiftsStartAndEndTogether(t *testing.T) {
	text := "ーち\n仕事\n08:30\n17:30\n？？\n毎日\n5\n\n2026-09-07\n？？"
	var timeiss []*kftlTimeIsRequest
	for _, r := range helperExpandOK(t, text) {
		if ti, ok := r.(*kftlTimeIsRequest); ok {
			timeiss = append(timeiss, ti)
		}
	}
	if len(timeiss) != 5 {
		t.Fatalf("件数 = %d, want 5", len(timeiss))
	}
	ids := map[string]struct{}{}
	for i, ti := range timeiss {
		day := 7 + i
		if ti.startTime.Year() != 2026 {
			t.Errorf("[%d] 開始の年 = %d, want 2026", i, ti.startTime.Year())
		}
		if want := ymdhm(2026, 9, day, 8, 30); !ti.startTime.Equal(want) {
			t.Errorf("[%d] 開始 = %v, want %v", i, ti.startTime, want)
		}
		if want := ymdhm(2026, 9, day, 17, 30); ti.endTime == nil || !ti.endTime.Equal(want) {
			t.Errorf("[%d] 終了 = %v, want %v", i, ti.endTime, want)
		}
		if ti.title != "仕事" {
			t.Errorf("[%d] title = %q", i, ti.title)
		}
		ids[ti.RequestID] = struct{}{}
	}
	if len(ids) != 5 {
		t.Errorf("IDは回ごとに別のはず: %d 種", len(ids))
	}
}

func TestExpand_KmemoRepeatUsesRelatedTime(t *testing.T) {
	text := "今日の日記\n？？\n毎日\n3\n？？"
	var kmemos []*kftlKmemoRequest
	for _, r := range helperExpandOK(t, text) {
		if k, ok := r.(*kftlKmemoRequest); ok {
			kmemos = append(kmemos, k)
		}
	}
	if len(kmemos) != 3 {
		t.Fatalf("件数 = %d, want 3", len(kmemos))
	}
	want := []time.Time{ymdhm(2026, 9, 3, 10, 0), ymdhm(2026, 9, 4, 10, 0), ymdhm(2026, 9, 5, 10, 0)}
	for i := range kmemos {
		if got := kmemos[i].GetRelatedTime(); !got.Equal(want[i]) {
			t.Errorf("[%d] 関連時刻 = %v, want %v", i, got, want[i])
		}
	}
}

// タグとテキストも回数ぶん複製される。テキストIDは回ごとに採り直す
// （使い回すと append-only なので最後の1件以外が消える）。
func TestExpand_RepeatClonesTagsAndTextsWithFreshTextIDs(t *testing.T) {
	text := "今日の日記\n。日記\nーー\n本文\nーー\n？？\n毎日\n2\n？？"
	var kmemos []*kftlKmemoRequest
	for _, r := range helperExpandOK(t, text) {
		if k, ok := r.(*kftlKmemoRequest); ok {
			kmemos = append(kmemos, k)
		}
	}
	if len(kmemos) != 2 {
		t.Fatalf("件数 = %d, want 2", len(kmemos))
	}
	seen := map[string]bool{}
	for i, k := range kmemos {
		if len(k.GetTags()) != 1 || k.GetTags()[0] != "日記" {
			t.Errorf("[%d] タグ = %v", i, k.GetTags())
		}
		if len(k.GetTextsMap()) != 1 {
			t.Fatalf("[%d] テキスト数 = %d, want 1", i, len(k.GetTextsMap()))
		}
		for textID := range k.GetTextsMap() {
			if seen[textID] {
				t.Errorf("テキストIDが複製で使い回されている: %s", textID)
			}
			seen[textID] = true
		}
	}
}

// related_time が主軸の残り3型（気分値・数値・ブックマーク）。
// 2083 年の事故（打刻）のあとの全型監査で、この3型には繰り返しのテストが1本も無かった。
// 本体の関連時刻と、タグ・テキストに使う基底の関連時刻（doBaseRequest へ渡す値）の両方を年まで見る。
// TS 側の対: kftl-repeat-statement.test.ts「繰り返しで書き込まれる時刻」
func TestExpand_RelatedTimeTypesShiftTogether(t *testing.T) {
	cases := []struct {
		name string
		text string
		pick func(KFTLRequest) (KFTLRequest, bool)
	}{
		{"気分値", "ーら\n5\n。気分\n？？\n毎日\n3\n？？", func(r KFTLRequest) (KFTLRequest, bool) {
			v, ok := r.(*kftlLantanaRequest)
			return v, ok
		}},
		{"数値", "ーか\n体重\n60\n。健康\n？？\n毎日\n3\n？？", func(r KFTLRequest) (KFTLRequest, bool) {
			v, ok := r.(*kftlKCRequest)
			return v, ok
		}},
		{"ブックマーク", "ーう\nhttps://example.com/\n例\n。ブックマーク\n？？\n毎日\n3\n？？", func(r KFTLRequest) (KFTLRequest, bool) {
			v, ok := r.(*kftlURLogRequest)
			return v, ok
		}},
	}
	want := []time.Time{ymdhm(2026, 9, 3, 10, 0), ymdhm(2026, 9, 4, 10, 0), ymdhm(2026, 9, 5, 10, 0)}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var picked []KFTLRequest
			for _, r := range helperExpandOK(t, c.text) {
				if v, ok := c.pick(r); ok {
					picked = append(picked, v)
				}
			}
			if len(picked) != 3 {
				t.Fatalf("件数 = %d, want 3", len(picked))
			}
			for i, r := range picked {
				got := r.GetRelatedTime()
				if got.Year() != 2026 {
					t.Errorf("[%d] 関連時刻の年 = %d, want 2026", i, got.Year())
				}
				if !got.Equal(want[i]) {
					t.Errorf("[%d] 関連時刻 = %v, want %v", i, got, want[i])
				}
				if len(r.GetTags()) != 1 {
					t.Errorf("[%d] タグ = %v, want 1件", i, r.GetTags())
				}
			}
		})
	}
}

// 支出ブロックの `？`行の時刻は、複製でもタグ・テキストに使う関連時刻（GetRelatedTime の override）に乗る。
// doBaseRequest が埋め込み基底の GetRelatedTime を引いていた頃は override が効かず、
// タグ・テキストだけ「今」で書かれていた。引数で渡す配線そのものは構造体からは見えないので、
// 書き込みまで通した検証は gkill_server_api の TestHandleSubmitKFTLText_RepeatWritesShiftedTimes にある
func TestExpand_NlogRepeatKeepsBlockTimeForTags(t *testing.T) {
	text := "ーん\nスーパー\n牛乳\n200\n。食費\n？2026-09-07 12:00\n？？\n金\n3\n？？"
	var nlogs []*kftlNlogRequest
	for _, r := range helperExpandOK(t, text) {
		if n, ok := r.(*kftlNlogRequest); ok {
			nlogs = append(nlogs, n)
		}
	}
	if len(nlogs) != 3 {
		t.Fatalf("件数 = %d, want 3", len(nlogs))
	}
	want := []time.Time{ymdhm(2026, 9, 4, 12, 0), ymdhm(2026, 9, 11, 12, 0), ymdhm(2026, 9, 18, 12, 0)}
	for i, n := range nlogs {
		// 外側の型で引く。doBaseRequest へ渡すのもこの値
		if got := n.GetRelatedTime(); !got.Equal(want[i]) {
			t.Errorf("[%d] タグ・テキストに使う関連時刻 = %v, want %v", i, got, want[i])
		}
		if len(n.GetTags()) != 1 || n.GetTags()[0] != "食費" {
			t.Errorf("[%d] タグ = %v", i, n.GetTags())
		}
	}
}

// 「～～」に揃える。閉じ忘れても必須2行が揃っていれば有効。
func TestExpand_UnclosedRepeatStillExpands(t *testing.T) {
	text := "今日の日記\n？？\n毎日\n3"
	count := 0
	for _, r := range helperExpandOK(t, text) {
		if _, ok := r.(*kftlKmemoRequest); ok {
			count++
		}
	}
	if count != 3 {
		t.Errorf("件数 = %d, want 3", count)
	}
}

// 4行を書き終えたあとの位置は受け皿。空行は見逃してブロックの中に留まるので、
// 空行を挟んでから閉じても件数は変わらない（クライアントはこの位置に「**********」の行ラベルを出す）。
func TestExpand_RepeatBlankLineAfterFieldsIsIgnored(t *testing.T) {
	text := "今日の日記\n？？\n毎日\n3\nno\n2026-09-10\n\n？？"
	count := 0
	for _, r := range helperExpandOK(t, text) {
		if _, ok := r.(*kftlKmemoRequest); ok {
			count++
		}
	}
	if count != 3 {
		t.Errorf("件数 = %d, want 3", count)
	}
	lines := helperGenerateLinesAt(t, text, repeatTestBase())
	if got := lines[6].GetLabelName(); got != "none" {
		t.Errorf("空行のラベル = %s, want none", got)
	}
	if got := lines[7].GetLabelName(); got != "endRepeat" {
		t.Errorf("閉じる行のラベル = %s, want endRepeat", got)
	}
}

func TestApply_RepeatRejectsForbiddenTypes(t *testing.T) {
	cases := []struct{ name, text string }{
		{"打刻開始のみ", "ーた\n仕事\n？？\n金\n3\n？？"},
		{"打刻終了", "ーえ\n仕事\n？？\n金\n3\n？？"},
		{"タグ指定の打刻終了", "ーたえ\n仕事\n？？\n金\n3\n？？"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := helperExpand(t, c.text)
			assertInputErrorID(t, err, "KFTL_REPEAT_TYPE_NOT_SUPPORTED_MESSAGE_TITLE")
		})
	}
}

func TestApply_RepeatWithoutTargetIsError(t *testing.T) {
	_, err := helperExpand(t, "？？\n金\n3\n？？")
	assertInputErrorID(t, err, "KFTL_REPEAT_NO_TARGET_MESSAGE_TITLE")
}

// Mi は related_time を持たないので、予定日時が1つも無いと繰り返しの入れ先が無い。
func TestExpand_MiWithoutDateFieldIsError(t *testing.T) {
	_, err := helperExpand(t, "ーみ\n週報\n仕事\n？？\n金\n3\n？？")
	assertInputErrorID(t, err, "KFTL_REPEAT_NO_DATE_FIELD_MESSAGE_TITLE")
}

// 4行を書き終えたあとは閉じる行しか来られない（飲み込むと本文が繰り返し指定に化ける）。
func TestApply_RepeatBlockWithExtraLineIsError(t *testing.T) {
	_, err := helperExpand(t, "今日の日記\n？？\n毎日\n3\nno\n2026-09-10\nゴミ\n？？")
	assertInputErrorID(t, err, "KFTL_REPEAT_NOT_CLOSED_MESSAGE_TITLE")
}

// 「？？ 金 3」は関連時刻行へ流さず、記号の書き方の行エラーにする。
func TestApply_RepeatWithArgumentOnSameLineIsError(t *testing.T) {
	_, err := helperExpand(t, "今日の日記\n？？ 金 3")
	assertInputErrorID(t, err, "KFTL_PREFIX_MUST_BE_ALONE_ON_LINE_MESSAGE_TITLE")
}

// 支出ブロックの店名・最初の品名の位置には書けない（次の行が固定で先読みが効かない）。
func TestApply_NlogRepeatBeforeFirstPaymentIsError(t *testing.T) {
	_, err := helperExpand(t, "ーん\n？？\n金\n3\n？？")
	assertInputErrorID(t, err, "KFTL_NLOG_META_INFO_MUST_BE_AFTER_AMOUNT_MESSAGE_TITLE")
}

func TestExpand_MiReKyouRepeat(t *testing.T) {
	text := "牛乳を買う\n～～\n仕事\n18:00\n？？\n金\n2\n？？\n～～"
	var mirekyous []*kftlMiReKyouRequest
	for _, r := range helperExpandOK(t, text) {
		if m, ok := r.(*kftlMiReKyouRequest); ok {
			mirekyous = append(mirekyous, m)
		}
	}
	if len(mirekyous) != 2 {
		t.Fatalf("件数 = %d, want 2", len(mirekyous))
	}
	// タスク化される元の記録のほうは繰り返さない。
	// ブロックの中の target_id は元の記録を指しているので、付け先を間違えるとこちらが増える
	kmemos := 0
	for _, r := range helperExpandOK(t, text) {
		if _, ok := r.(*kftlKmemoRequest); ok {
			kmemos++
		}
	}
	if kmemos != 1 {
		t.Errorf("元の記録の件数 = %d, want 1", kmemos)
	}
	want := []time.Time{ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 9, 11, 18, 0)}
	for i := range mirekyous {
		if mirekyous[i].estimateStartTime == nil || !mirekyous[i].estimateStartTime.Equal(want[i]) {
			t.Errorf("[%d] 見積開始 = %v, want %v", i, mirekyous[i].estimateStartTime, want[i])
		}
		// 対象は同じ記録を指したまま（同じ記録を繰り返しタスク化する）
		if mirekyous[i].targetID != mirekyous[0].targetID {
			t.Errorf("[%d] 対象IDが変わっている", i)
		}
	}
}

// ─── 既存スキップ（3行目 no）─────────────────────────────────────────────────

// repeatMockRequest は既存判定の結果を差し替えられるリクエスト。
// 実リポジトリを立てずに「既にある回を飛ばす」ところだけを固定するために使う。
type repeatMockRequest struct {
	KFTLRequestBase
	existing map[int64]struct{}
}

func (r *repeatMockRequest) DoRequest(_ context.Context) error { return nil }

// ValidateContent も基底に既定実装が無い（書き忘れをコンパイルで捕まえるため）ので、モックにも要る。
func (r *repeatMockRequest) ValidateContent() error { return nil }

func (r *repeatMockRequest) CloneForRepeat(newRequestID string, dayShift int) KFTLRequest {
	c := *r
	c.KFTLRequestBase = r.cloneBase(newRequestID, dayShift)
	return &c
}

func (r *repeatMockRequest) FindExistingForRepeat(_ context.Context, _, _ time.Time) (map[int64]struct{}, error) {
	return r.existing, nil
}

func helperRepeatMock(t *testing.T, id string, anchor time.Time, existing map[int64]struct{}, condText string, count int, addIfExists bool) []KFTLRequest {
	t.Helper()
	base := repeatTestBase()
	req := &repeatMockRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  id,
			Ctx:        &KFTLStatementLineContext{},
			CreateTime: anchor,
		},
		existing: existing,
	}
	cond, err := parseRepeatCondition(condText)
	if err != nil {
		t.Fatalf("parseRepeatCondition: %v", err)
	}
	if err := req.SetRepeatSpec(&repeatSpec{cond: cond, count: count, origin: &base, addIfExists: addIfExists}); err != nil {
		t.Fatalf("SetRepeatSpec: %v", err)
	}
	requestMap := NewKFTLRequestMap()
	if err := requestMap.Set(id, req); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := expandRepeats(context.Background(), requestMap, base); err != nil {
		t.Fatalf("expandRepeats: %v", err)
	}
	return requestMap.All()
}

// 既定（3行目 no）は既にある回を飛ばす。同じテキストを何度送っても増えない。
func TestExpand_SkipsOccurrencesThatAlreadyExist(t *testing.T) {
	anchor := ymdhm(2026, 9, 2, 18, 0)
	existing := map[int64]struct{}{ymdhm(2026, 9, 11, 18, 0).Unix(): {}}
	got := helperRepeatMock(t, "mock-1", anchor, existing, "金", 3, false)
	if len(got) != 2 {
		t.Fatalf("件数 = %d, want 2（9/11 は既にあるので飛ばす）", len(got))
	}
	want := []time.Time{ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 9, 18, 18, 0)}
	for i, r := range got {
		if rt := r.GetRelatedTime(); !rt.Equal(want[i]) {
			t.Errorf("[%d] = %v, want %v", i, rt, want[i])
		}
	}
	// 元のIDを引き継ぐのは「実際に作る最初の1件」。先頭の回が飛んでも引き継がれる
	if got[0].GetRequestID() != "mock-1" {
		t.Errorf("先頭のID = %q, want mock-1", got[0].GetRequestID())
	}
}

// 先頭の回が飛ばされても、作られた1件目が元のIDを引き継ぐ。
func TestExpand_FirstCreatedKeepsOriginalIDEvenIfEarlierSkipped(t *testing.T) {
	anchor := ymdhm(2026, 9, 2, 18, 0)
	existing := map[int64]struct{}{ymdhm(2026, 9, 4, 18, 0).Unix(): {}}
	got := helperRepeatMock(t, "mock-1", anchor, existing, "金", 2, false)
	if len(got) != 1 {
		t.Fatalf("件数 = %d, want 1", len(got))
	}
	if got[0].GetRequestID() != "mock-1" {
		t.Errorf("ID = %q, want mock-1", got[0].GetRequestID())
	}
	if rt := got[0].GetRelatedTime(); !rt.Equal(ymdhm(2026, 9, 11, 18, 0)) {
		t.Errorf("関連時刻 = %v, want 2026-09-11 18:00", rt)
	}
}

// 3行目 yes は既存を見ずに全部作る。
func TestExpand_AddIfExistsCreatesAll(t *testing.T) {
	anchor := ymdhm(2026, 9, 2, 18, 0)
	existing := map[int64]struct{}{ymdhm(2026, 9, 11, 18, 0).Unix(): {}}
	got := helperRepeatMock(t, "mock-1", anchor, existing, "金", 3, true)
	if len(got) != 3 {
		t.Errorf("件数 = %d, want 3（yes は既存を見ない）", len(got))
	}
}

// 全部の回が既にあれば0件。再送しても増えないという冪等性の下限。
func TestExpand_AllOccurrencesExistProducesNothing(t *testing.T) {
	anchor := ymdhm(2026, 9, 2, 18, 0)
	existing := map[int64]struct{}{
		ymdhm(2026, 9, 4, 18, 0).Unix():  {},
		ymdhm(2026, 9, 11, 18, 0).Unix(): {},
	}
	got := helperRepeatMock(t, "mock-1", anchor, existing, "金", 2, false)
	if len(got) != 0 {
		t.Errorf("件数 = %d, want 0", len(got))
	}
}

// アンカーの見方が既存判定と展開でずれると噛み合わない。同じ順序であることを固定する。
func TestScheduleAnchorOfMatchesMiAnchor(t *testing.T) {
	start := ymdhm(2026, 9, 4, 9, 0)
	end := ymdhm(2026, 9, 4, 10, 0)
	limit := ymdhm(2026, 9, 5, 18, 0)
	cases := []struct {
		name                    string
		estStart, estEnd, limit *time.Time
		want                    *time.Time
	}{
		{"見積開始が最優先", &start, &end, &limit, &start},
		{"見積開始が無ければ見積終了", nil, &end, &limit, &end},
		{"どちらも無ければ期限", nil, nil, &limit, &limit},
		{"1つも無ければ入れ先なし", nil, nil, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mi := &kftlMiRequest{estimateStartTime: c.estStart, estimateEndTime: c.estEnd, limitTime: c.limit}
			gotAnchor, gotOK := mi.AnchorTimeForRepeat()
			wantAnchor, wantOK := scheduleAnchorOf(c.estStart, c.estEnd, c.limit)
			if gotOK != wantOK || (gotOK && !gotAnchor.Equal(wantAnchor)) {
				t.Errorf("AnchorTimeForRepeat=(%v,%v) scheduleAnchorOf=(%v,%v)", gotAnchor, gotOK, wantAnchor, wantOK)
			}
			if c.want == nil {
				if gotOK {
					t.Errorf("入れ先なしのはず: %v", gotAnchor)
				}
				return
			}
			if !gotAnchor.Equal(*c.want) {
				t.Errorf("anchor = %v, want %v", gotAnchor, *c.want)
			}
		})
	}
}

// 「？？」のカーブアウトは「完全一致」と「直後が空白（引数つき）」の2つだけ。
// 「??なんだこれ」のように直後が空白でない行は、従来どおり「？」の前方一致へ流れて
// 日時として解釈され、日時でなければ不正行になる（`？` 始まりの行の既存仕様を変えない）。
// クライアント側と揃えてある。
func TestApply_RepeatPrefixWithoutSpaceStaysRelatedTime(t *testing.T) {
	_, err := helperExpand(t, "今日の日記\n??なんだこれ")
	assertInputErrorID(t, err, "KFTL_INVALID_PARSE_RELATED_TIME_ERROR_MESSAGE_TITLE")
}
