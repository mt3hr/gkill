package kftl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 繰り返しブロック「？？」の仕様。行の解釈と展開は kftl_repeat_lines.go 側。
//
// **展開（クローン生成）をここでやらないこと。** 候補日時の計算は上限つきで軽いが、
// レコードの複製は送信時（GenerateAndExecuteRequests の実行ループ直前）に回す。
// クライアント側は本文が変わるたびに全行の ApplyThisLineToRequestMap を回すので、
// 解釈のフェーズで複製すると打鍵1回あたり最大 repeatMaxRecords 件を作ることになる。

const (
	// repeatMaxRecords は1つの「？？」ブロック、および1回の送信が作れるレコード数の上限。
	// 超えたら黙って切り詰めず入力エラーにする（KFTL の書き込みは部分確定するので、
	// 大量生成が途中で失敗すると半端に残る）。
	repeatMaxRecords = 1000
	// repeatScanDays / repeatScanMonths は候補の走査打ち切り。
	// 条件に一致する日が一度も来ない書き方をしたときに無限に回らないようにする。
	repeatScanDays   = 3653 // 約10年
	repeatScanMonths = 120  // 約10年
)

// repeatCondKind は繰り返し条件の種類。
type repeatCondKind int

const (
	repeatCondDaily      repeatCondKind = iota // 毎日
	repeatCondWeekday                          // 曜日（N週おき）
	repeatCondNthWeekday                       // 第N曜日 / 最終曜日
	repeatCondMonthDay                         // 毎月N日
)

// repeatCond は「どの日か」。
type repeatCond struct {
	kind repeatCondKind
	// weekdays は repeatCondWeekday で使う（複数可）。repeatCondNthWeekday では1つだけ。
	weekdays []time.Weekday
	// weekInterval は repeatCondWeekday の「N週おき」。1 が毎週。
	// 2以上のときは weekdays をちょうど1つに絞る（「2週月水」は基準週が決められない）。
	weekInterval int
	// nth は repeatCondNthWeekday の第N。-1 は最終。
	nth int
	// monthDay は repeatCondMonthDay の日。
	monthDay int
}

// repeatSpec は「？？」ブロック1つぶんの指定。行が順に埋めていく可変オブジェクトで、
// Nlog ブロックのように複数のリクエストが同じポインタを共有することがある
// （共有しているものが1つの繰り返しグループになる）。
type repeatSpec struct {
	cond *repeatCond // 1行目。必須
	// count は2行目が回数のとき。until は2行目が終了日のとき。どちらか一方が入る。
	count int
	until *time.Time
	// addIfExists は3行目。既定 false（既存があればその回を作らない）。
	addIfExists bool
	// origin は4行目。nil なら BaseTime。
	origin *time.Time

	// lineIndex / lineText は開始行の位置。展開は実行ループの直前なので、
	// 控えておかないと展開で落ちたときに応答から原因の行が分からない。
	lineIndex int
	lineText  string
}

// ─── 1行目（条件）のパース ────────────────────────────────────────────────────

var jaWeekdays = map[rune]time.Weekday{
	'日': time.Sunday, '月': time.Monday, '火': time.Tuesday, '水': time.Wednesday,
	'木': time.Thursday, '金': time.Friday, '土': time.Saturday,
}

var asciiWeekdays = map[string]time.Weekday{
	"sun": time.Sunday, "mon": time.Monday, "tue": time.Tuesday, "wed": time.Wednesday,
	"thu": time.Thursday, "fri": time.Friday, "sat": time.Saturday,
}

var (
	reJaNthWeekday   = regexp.MustCompile(`^第([1-5])(.)$`)
	reJaWeekInterval = regexp.MustCompile(`^([1-9][0-9]*)週(.+)$`)
	reJaMonthDay     = regexp.MustCompile(`^毎月([1-9][0-9]*)$`)
	reJaLastWeekday  = regexp.MustCompile(`^最終(.)$`)

	reAsNthWeekday   = regexp.MustCompile(`^([1-5])(?:st|nd|rd|th)\s+([a-z]{3})$`)
	reAsWeekInterval = regexp.MustCompile(`^([1-9][0-9]*)w\s+([a-z]{3})$`)
	reAsMonthDay     = regexp.MustCompile(`^monthly\s+([1-9][0-9]*)$`)
	reAsLastWeekday  = regexp.MustCompile(`^last\s+([a-z]{3})$`)
)

// parseRepeatCondition は「？？」ブロックの1行目を読む。
// 日本語と ASCII の両方を受ける（書式の対応表は kftl/README.md）。
func parseRepeatCondition(lineText string) (*repeatCond, error) {
	s := strings.TrimSpace(lineText)
	if s == "" {
		return nil, newKFTLInputError("KFTL_REPEAT_CONDITION_REQUIRED_MESSAGE_TITLE",
			fmt.Errorf("repeat condition is empty"))
	}
	if cond := parseJaRepeatCondition(s); cond != nil {
		return cond, nil
	}
	if cond := parseAsciiRepeatCondition(strings.ToLower(s)); cond != nil {
		return cond, nil
	}
	return nil, newKFTLInputError("KFTL_REPEAT_INVALID_CONDITION_MESSAGE_TITLE",
		fmt.Errorf("cannot parse repeat condition: %q", lineText))
}

func parseJaRepeatCondition(s string) *repeatCond {
	if s == "毎日" {
		return &repeatCond{kind: repeatCondDaily}
	}
	if m := reJaLastWeekday.FindStringSubmatch(s); m != nil {
		if wd, ok := jaWeekdayOf(m[1]); ok {
			return &repeatCond{kind: repeatCondNthWeekday, weekdays: []time.Weekday{wd}, nth: -1}
		}
		return nil
	}
	if m := reJaNthWeekday.FindStringSubmatch(s); m != nil {
		if wd, ok := jaWeekdayOf(m[2]); ok {
			n, _ := strconv.Atoi(m[1])
			return &repeatCond{kind: repeatCondNthWeekday, weekdays: []time.Weekday{wd}, nth: n}
		}
		return nil
	}
	if m := reJaMonthDay.FindStringSubmatch(s); m != nil {
		day, _ := strconv.Atoi(m[1])
		if day < 1 || day > 31 {
			return nil
		}
		return &repeatCond{kind: repeatCondMonthDay, monthDay: day}
	}
	if m := reJaWeekInterval.FindStringSubmatch(s); m != nil {
		wds, ok := jaWeekdaysOf(m[2])
		// 「N週」は基準の週を1つに決める必要があるので、曜日はちょうど1つ。
		if !ok || len(wds) != 1 {
			return nil
		}
		n, _ := strconv.Atoi(m[1])
		return &repeatCond{kind: repeatCondWeekday, weekdays: wds, weekInterval: n}
	}
	if wds, ok := jaWeekdaysOf(s); ok {
		return &repeatCond{kind: repeatCondWeekday, weekdays: wds, weekInterval: 1}
	}
	return nil
}

func parseAsciiRepeatCondition(s string) *repeatCond {
	if s == "daily" {
		return &repeatCond{kind: repeatCondDaily}
	}
	if m := reAsLastWeekday.FindStringSubmatch(s); m != nil {
		if wd, ok := asciiWeekdays[m[1]]; ok {
			return &repeatCond{kind: repeatCondNthWeekday, weekdays: []time.Weekday{wd}, nth: -1}
		}
		return nil
	}
	if m := reAsNthWeekday.FindStringSubmatch(s); m != nil {
		if wd, ok := asciiWeekdays[m[2]]; ok {
			n, _ := strconv.Atoi(m[1])
			return &repeatCond{kind: repeatCondNthWeekday, weekdays: []time.Weekday{wd}, nth: n}
		}
		return nil
	}
	if m := reAsMonthDay.FindStringSubmatch(s); m != nil {
		day, _ := strconv.Atoi(m[1])
		if day < 1 || day > 31 {
			return nil
		}
		return &repeatCond{kind: repeatCondMonthDay, monthDay: day}
	}
	if m := reAsWeekInterval.FindStringSubmatch(s); m != nil {
		wd, ok := asciiWeekdays[m[2]]
		if !ok {
			return nil
		}
		n, _ := strconv.Atoi(m[1])
		return &repeatCond{kind: repeatCondWeekday, weekdays: []time.Weekday{wd}, weekInterval: n}
	}
	var wds []time.Weekday
	seen := map[time.Weekday]bool{}
	for _, part := range strings.Split(s, ",") {
		wd, ok := asciiWeekdays[strings.TrimSpace(part)]
		if !ok {
			return nil
		}
		if !seen[wd] {
			seen[wd] = true
			wds = append(wds, wd)
		}
	}
	if len(wds) == 0 {
		return nil
	}
	return &repeatCond{kind: repeatCondWeekday, weekdays: wds, weekInterval: 1}
}

func jaWeekdayOf(s string) (time.Weekday, bool) {
	runes := []rune(s)
	if len(runes) != 1 {
		return 0, false
	}
	wd, ok := jaWeekdays[runes[0]]
	return wd, ok
}

// jaWeekdaysOf は「月水金」のような並びを曜日の集合にする。重複は畳む。
func jaWeekdaysOf(s string) ([]time.Weekday, bool) {
	var wds []time.Weekday
	seen := map[time.Weekday]bool{}
	for _, r := range s {
		wd, ok := jaWeekdays[r]
		if !ok {
			return nil, false
		}
		if !seen[wd] {
			seen[wd] = true
			wds = append(wds, wd)
		}
	}
	if len(wds) == 0 {
		return nil, false
	}
	return wds, true
}

// ─── 2行目（回数 または 終了日）のパース ───────────────────────────────────────

// parseRepeatCountOrUntil は2行目を読む。整数だけなら回数、日時として読めれば終了日。
// 整数が先。parseDateTime は裸の整数の書式を持たないので曖昧にならない。
//
// 終了日が日付のみ（時刻を含まない）なら、その日の 23:59:59 まで伸ばす
// （FindQuery の calendar_end_date と同じ流儀。伸ばさないと当日の 18:00 が範囲外になる）。
func parseRepeatCountOrUntil(lineText string, base time.Time) (int, *time.Time, error) {
	s := strings.TrimSpace(lineText)
	if s == "" {
		return 0, nil, newKFTLInputError("KFTL_REPEAT_COUNT_REQUIRED_MESSAGE_TITLE",
			fmt.Errorf("repeat needs a count or an end date"))
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n < 1 || n > repeatMaxRecords {
			return 0, nil, newKFTLInputError("KFTL_REPEAT_LIMIT_EXCEEDED_MESSAGE_TITLE",
				fmt.Errorf("repeat count %d is out of range (1..%d)", n, repeatMaxRecords))
		}
		return n, nil, nil
	}
	t, err := parseDateTime(s, base)
	if err != nil {
		return 0, nil, newKFTLInputError("KFTL_REPEAT_INVALID_COUNT_MESSAGE_TITLE",
			fmt.Errorf("repeat second line is neither a count nor a date: %q", lineText))
	}
	if !strings.Contains(s, ":") {
		t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
	}
	return 0, &t, nil
}

// ─── 3行目（既存があっても追加するか）のパース ─────────────────────────────────

// parseRepeatAddIfExists は3行目を読む。空は既定の「追加しない」。
// 既定を「追加しない」にしてあるので、同じテキストを何度送っても増えない（冪等）。
func parseRepeatAddIfExists(lineText string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(lineText)) {
	case "", "no", "いいえ":
		return false, nil
	case "yes", "はい":
		return true, nil
	}
	return false, newKFTLInputError("KFTL_REPEAT_INVALID_DUPLICATE_MESSAGE_TITLE",
		fmt.Errorf("repeat third line must be yes or no: %q", lineText))
}

// ─── 4行目（起点）のパース ────────────────────────────────────────────────────

// parseRepeatOrigin は4行目を読む。空なら nil（呼び出し側が BaseTime を使う）。
// 書式は関連時刻と同じものを全部受ける。
func parseRepeatOrigin(lineText string, base time.Time) (*time.Time, error) {
	s := strings.TrimSpace(lineText)
	if s == "" {
		return nil, nil
	}
	t, err := parseDateTime(s, base)
	if err != nil {
		return nil, newKFTLInputError("KFTL_REPEAT_INVALID_ORIGIN_MESSAGE_TITLE",
			fmt.Errorf("cannot parse repeat origin %q: %w", lineText, err))
	}
	return &t, nil
}

// ─── 候補日時の計算 ───────────────────────────────────────────────────────────

// occurrencesOf は条件に一致する日時を順に返す。
//
// 時刻はアンカー（そのレコードが持つ日時欄のうち最初に埋まっているもの）から取る。
// **起点ちょうどは含めない**（`候補 > 起点`）。
//
// 走査は repeatScanDays / repeatScanMonths で打ち切る。条件に一致する日が
// 一度も来ない書き方をしたときに回り続けないようにするためで、
// 打ち切りに達しても見つかったぶんをそのまま返す（0件は呼び出し側が弾く）。
func occurrencesOf(spec *repeatSpec, anchor time.Time, base time.Time) ([]time.Time, error) {
	if spec == nil || spec.cond == nil {
		return nil, newKFTLInputError("KFTL_REPEAT_CONDITION_REQUIRED_MESSAGE_TITLE",
			fmt.Errorf("repeat condition is not set"))
	}
	origin := base
	if spec.origin != nil {
		origin = *spec.origin
	}
	hour, minute, sec := anchor.Clock()
	at := func(d time.Time) time.Time {
		return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, sec, 0, time.Local)
	}

	var out []time.Time
	var acceptErr error
	// accept は候補を1つ受け取り、走査を続けてよいかを返す。
	accept := func(c time.Time) bool {
		if !c.After(origin) {
			return true // 起点以前。まだ先に候補がある
		}
		if spec.until != nil && c.After(*spec.until) {
			return false
		}
		out = append(out, c)
		if len(out) > repeatMaxRecords {
			acceptErr = newKFTLInputError("KFTL_REPEAT_LIMIT_EXCEEDED_MESSAGE_TITLE",
				fmt.Errorf("repeat produces more than %d records", repeatMaxRecords))
			return false
		}
		return !(spec.count > 0 && len(out) >= spec.count)
	}

	switch {
	case spec.cond.kind == repeatCondNthWeekday || spec.cond.kind == repeatCondMonthDay:
		cur := time.Date(origin.Year(), origin.Month(), 1, 0, 0, 0, 0, time.Local)
		for i := 0; i <= repeatScanMonths; i++ {
			if d, ok := monthlyCandidate(spec.cond, cur); ok {
				if !accept(at(d)) {
					break
				}
			}
			cur = cur.AddDate(0, 1, 0)
		}
	case spec.cond.kind == repeatCondWeekday && spec.cond.weekInterval > 1:
		// N週おきは「最初の一致」を基準にして、そこから N 週ずつ送る。
		// 日送りで拾うと基準週が決まらない。
		wd := spec.cond.weekdays[0]
		cur := time.Date(origin.Year(), origin.Month(), origin.Day(), 0, 0, 0, 0, time.Local)
		found := false
		for i := 0; i < 8; i++ { // 今日がその曜日でも時刻が過ぎていれば翌週まで送る
			if cur.Weekday() == wd && at(cur).After(origin) {
				found = true
				break
			}
			cur = cur.AddDate(0, 0, 1)
		}
		if found {
			step := 7 * spec.cond.weekInterval
			for i := 0; i*step <= repeatScanDays; i++ {
				if !accept(at(cur)) {
					break
				}
				cur = cur.AddDate(0, 0, step)
			}
		}
	default:
		cur := time.Date(origin.Year(), origin.Month(), origin.Day(), 0, 0, 0, 0, time.Local)
		for i := 0; i <= repeatScanDays; i++ {
			if matchesDailyOrWeekday(spec.cond, cur) {
				if !accept(at(cur)) {
					break
				}
			}
			cur = cur.AddDate(0, 0, 1)
		}
	}
	if acceptErr != nil {
		return nil, acceptErr
	}
	return out, nil
}

func matchesDailyOrWeekday(cond *repeatCond, d time.Time) bool {
	if cond.kind == repeatCondDaily {
		return true
	}
	for _, wd := range cond.weekdays {
		if d.Weekday() == wd {
			return true
		}
	}
	return false
}

// monthlyCandidate は monthStart の月の候補日を返す。
//
// **存在しない日はその月を飛ばす**（第5金が無い月、2月の「毎月31」など）。
// 最も近い日へ丸めると「第4金」「毎月30」と重複するので、丸めない。
func monthlyCandidate(cond *repeatCond, monthStart time.Time) (time.Time, bool) {
	year, month := monthStart.Year(), monthStart.Month()
	switch cond.kind {
	case repeatCondMonthDay:
		d := time.Date(year, month, cond.monthDay, 0, 0, 0, 0, time.Local)
		if d.Month() != month {
			return time.Time{}, false // 繰り上がった = その月に無い日
		}
		return d, true
	case repeatCondNthWeekday:
		wd := cond.weekdays[0]
		if cond.nth == -1 {
			// 「月の0日」= 前月の末日。month+1 の0日で当月末日になる
			d := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local)
			for d.Weekday() != wd {
				d = d.AddDate(0, 0, -1)
			}
			return d, true
		}
		first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
		offset := (int(wd) - int(first.Weekday()) + 7) % 7
		d := first.AddDate(0, 0, offset+(cond.nth-1)*7)
		if d.Month() != month {
			return time.Time{}, false // 第5が無い月
		}
		return d, true
	}
	return time.Time{}, false
}
