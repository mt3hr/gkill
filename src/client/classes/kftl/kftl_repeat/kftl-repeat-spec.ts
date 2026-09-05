'use strict'

import { i18n } from '@/i18n'
import { parse_kftl_date_time } from '../kftl-date-time'

/**
 * 繰り返しブロック「？？」の仕様。行の解釈と展開は同じディレクトリの他のファイル。
 *
 * **展開（クローン生成）をここでやらないこと。** 候補日時の計算は上限つきで軽いが、
 * レコードの複製は送信時（generate_requests の最後）に回す。
 * `use-kftl-view.ts` は本文が変わるたびに全行の apply_this_line_to_request_map を回すので、
 * 解釈のフェーズで複製すると打鍵1回あたり最大 REPEAT_MAX_RECORDS 件を作ることになる。
 *
 * Mirrors: src/server/gkill/api/kftl/kftl_repeat.go
 */

/** 1つの「？？」ブロック、および1回の送信が作れるレコード数の上限。 */
export const REPEAT_MAX_RECORDS = 1000
/** 候補の走査打ち切り（約10年）。条件に一致する日が来ない書き方で回り続けないようにする。 */
const REPEAT_SCAN_DAYS = 3653
const REPEAT_SCAN_MONTHS = 120

export type RepeatCondKind = 'daily' | 'weekday' | 'nth_weekday' | 'month_day'

/** 「どの日か」。 */
export interface RepeatCond {
    kind: RepeatCondKind
    /** weekday で使う（複数可）。nth_weekday では1つだけ。 */
    weekdays: Array<number>
    /** weekday の「N週おき」。1 が毎週。2以上のときは weekdays をちょうど1つに絞る。 */
    week_interval: number
    /** nth_weekday の第N。-1 は最終。 */
    nth: number
    /** month_day の日。 */
    month_day: number
}

/**
 * 「？？」ブロック1つぶんの指定。行が順に埋めていく可変オブジェクトで、
 * 支出ブロックのように複数のリクエストが同じ参照を共有することがある
 * （共有しているものが1つの繰り返しグループになる）。
 */
export interface RepeatSpec {
    cond: RepeatCond | null
    /** 2行目が回数のとき。until と排他。 */
    count: number
    /** 2行目が終了日のとき。count と排他。 */
    until: Date | null
    /** 3行目。既定 false（既存があればその回を作らない）。 */
    add_if_exists: boolean
    /** 4行目。null なら送信時刻。 */
    origin: Date | null
    /** 開始行の位置。エラー表示のためだけに持つ。 */
    line_index: number
}

export function new_repeat_spec(line_index: number): RepeatSpec {
    return { cond: null, count: 0, until: null, add_if_exists: false, origin: null, line_index: line_index }
}

// ─── 1行目（条件）のパース ────────────────────────────────────────────────────

const JA_WEEKDAYS: Readonly<Record<string, number>> = {
    '日': 0, '月': 1, '火': 2, '水': 3, '木': 4, '金': 5, '土': 6,
}

const ASCII_WEEKDAYS: Readonly<Record<string, number>> = {
    sun: 0, mon: 1, tue: 2, wed: 3, thu: 4, fri: 5, sat: 6,
}

const RE_JA_NTH_WEEKDAY = /^第([1-5])(.)$/
const RE_JA_WEEK_INTERVAL = /^([1-9][0-9]*)週(.+)$/
const RE_JA_MONTH_DAY = /^毎月([1-9][0-9]*)$/
const RE_JA_LAST_WEEKDAY = /^最終(.)$/

const RE_AS_NTH_WEEKDAY = /^([1-5])(?:st|nd|rd|th)\s+([a-z]{3})$/
const RE_AS_WEEK_INTERVAL = /^([1-9][0-9]*)w\s+([a-z]{3})$/
const RE_AS_MONTH_DAY = /^monthly\s+([1-9][0-9]*)$/
const RE_AS_LAST_WEEKDAY = /^last\s+([a-z]{3})$/

function cond_of(kind: RepeatCondKind, weekdays: Array<number>, week_interval: number, nth: number, month_day: number): RepeatCond {
    return { kind: kind, weekdays: weekdays, week_interval: week_interval, nth: nth, month_day: month_day }
}

/**
 * 「？？」ブロックの1行目を読む。日本語と ASCII の両方を受ける。
 * 読めなければ例外（行がピンクになって保存も止まる）。
 */
export function parse_repeat_condition(line_text: string): RepeatCond {
    const text = line_text.trim()
    if (text === "") {
        throw new Error(i18n.global.t("KFTL_REPEAT_CONDITION_REQUIRED_MESSAGE_TITLE"))
    }
    const ja = parse_ja_repeat_condition(text)
    if (ja !== null) {
        return ja
    }
    const ascii = parse_ascii_repeat_condition(text.toLowerCase())
    if (ascii !== null) {
        return ascii
    }
    throw new Error(i18n.global.t("KFTL_REPEAT_INVALID_CONDITION_MESSAGE_TITLE"))
}

function parse_ja_repeat_condition(text: string): RepeatCond | null {
    if (text === "毎日") {
        return cond_of('daily', [], 0, 0, 0)
    }
    const last_matched = RE_JA_LAST_WEEKDAY.exec(text)
    if (last_matched !== null) {
        const weekday = ja_weekday_of(last_matched[1])
        return weekday === null ? null : cond_of('nth_weekday', [weekday], 0, -1, 0)
    }
    const nth_matched = RE_JA_NTH_WEEKDAY.exec(text)
    if (nth_matched !== null) {
        const weekday = ja_weekday_of(nth_matched[2])
        return weekday === null ? null : cond_of('nth_weekday', [weekday], 0, Number.parseInt(nth_matched[1]), 0)
    }
    const month_day_matched = RE_JA_MONTH_DAY.exec(text)
    if (month_day_matched !== null) {
        const day = Number.parseInt(month_day_matched[1])
        return day < 1 || day > 31 ? null : cond_of('month_day', [], 0, 0, day)
    }
    const interval_matched = RE_JA_WEEK_INTERVAL.exec(text)
    if (interval_matched !== null) {
        const weekdays = ja_weekdays_of(interval_matched[2])
        // 「N週」は基準の週を1つに決める必要があるので、曜日はちょうど1つ
        if (weekdays === null || weekdays.length !== 1) {
            return null
        }
        return cond_of('weekday', weekdays, Number.parseInt(interval_matched[1]), 0, 0)
    }
    const weekdays = ja_weekdays_of(text)
    return weekdays === null ? null : cond_of('weekday', weekdays, 1, 0, 0)
}

function parse_ascii_repeat_condition(text: string): RepeatCond | null {
    if (text === "daily") {
        return cond_of('daily', [], 0, 0, 0)
    }
    const last_matched = RE_AS_LAST_WEEKDAY.exec(text)
    if (last_matched !== null) {
        const weekday = ASCII_WEEKDAYS[last_matched[1]]
        return weekday === undefined ? null : cond_of('nth_weekday', [weekday], 0, -1, 0)
    }
    const nth_matched = RE_AS_NTH_WEEKDAY.exec(text)
    if (nth_matched !== null) {
        const weekday = ASCII_WEEKDAYS[nth_matched[2]]
        return weekday === undefined ? null : cond_of('nth_weekday', [weekday], 0, Number.parseInt(nth_matched[1]), 0)
    }
    const month_day_matched = RE_AS_MONTH_DAY.exec(text)
    if (month_day_matched !== null) {
        const day = Number.parseInt(month_day_matched[1])
        return day < 1 || day > 31 ? null : cond_of('month_day', [], 0, 0, day)
    }
    const interval_matched = RE_AS_WEEK_INTERVAL.exec(text)
    if (interval_matched !== null) {
        const weekday = ASCII_WEEKDAYS[interval_matched[2]]
        return weekday === undefined ? null : cond_of('weekday', [weekday], Number.parseInt(interval_matched[1]), 0, 0)
    }
    const weekdays: Array<number> = []
    const seen = new Set<number>()
    for (const part of text.split(",")) {
        const weekday = ASCII_WEEKDAYS[part.trim()]
        if (weekday === undefined) {
            return null
        }
        if (!seen.has(weekday)) {
            seen.add(weekday)
            weekdays.push(weekday)
        }
    }
    return weekdays.length === 0 ? null : cond_of('weekday', weekdays, 1, 0, 0)
}

function ja_weekday_of(text: string): number | null {
    const chars = Array.from(text)
    if (chars.length !== 1) {
        return null
    }
    const weekday = JA_WEEKDAYS[chars[0]]
    return weekday === undefined ? null : weekday
}

/** 「月水金」のような並びを曜日の集合にする。重複は畳む。 */
function ja_weekdays_of(text: string): Array<number> | null {
    const weekdays: Array<number> = []
    const seen = new Set<number>()
    for (const char of Array.from(text)) {
        const weekday = JA_WEEKDAYS[char]
        if (weekday === undefined) {
            return null
        }
        if (!seen.has(weekday)) {
            seen.add(weekday)
            weekdays.push(weekday)
        }
    }
    return weekdays.length === 0 ? null : weekdays
}

// ─── 2行目（回数 または 終了日）のパース ───────────────────────────────────────

export interface RepeatCountOrUntil {
    count: number
    until: Date | null
}

/**
 * 2行目を読む。整数だけなら回数、日時として読めれば終了日。整数が先。
 *
 * 終了日が日付のみ（時刻を含まない）なら、その日の 23:59:59 まで伸ばす
 * （FindQuery の calendar_end_date と同じ流儀。伸ばさないと当日の 18:00 が範囲外になる）。
 */
export function parse_repeat_count_or_until(line_text: string): RepeatCountOrUntil {
    const text = line_text.trim()
    if (text === "") {
        throw new Error(i18n.global.t("KFTL_REPEAT_COUNT_REQUIRED_MESSAGE_TITLE"))
    }
    if (/^[+-]?[0-9]+$/.test(text)) {
        const count = Number.parseInt(text)
        if (count < 1 || count > REPEAT_MAX_RECORDS) {
            throw new Error(i18n.global.t("KFTL_REPEAT_LIMIT_EXCEEDED_MESSAGE_TITLE"))
        }
        return { count: count, until: null }
    }
    const parsed = parse_kftl_date_time(text)
    if (parsed === null) {
        throw new Error(i18n.global.t("KFTL_REPEAT_INVALID_COUNT_MESSAGE_TITLE"))
    }
    if (!text.includes(":")) {
        return {
            count: 0,
            until: new Date(parsed.getFullYear(), parsed.getMonth(), parsed.getDate(), 23, 59, 59),
        }
    }
    return { count: 0, until: parsed }
}

// ─── 3行目（既存があっても追加するか）のパース ─────────────────────────────────

/**
 * 3行目を読む。空は既定の「追加しない」。
 * 既定を「追加しない」にしてあるので、同じテキストを何度送っても増えない（冪等）。
 */
export function parse_repeat_add_if_exists(line_text: string): boolean {
    switch (line_text.trim().toLowerCase()) {
        case "":
        case "no":
        case "いいえ":
            return false
        case "yes":
        case "はい":
            return true
    }
    throw new Error(i18n.global.t("KFTL_REPEAT_INVALID_DUPLICATE_MESSAGE_TITLE"))
}

// ─── 4行目（起点）のパース ────────────────────────────────────────────────────

/** 4行目を読む。空なら null（呼び出し側が送信時刻を使う）。 */
export function parse_repeat_origin(line_text: string): Date | null {
    const text = line_text.trim()
    if (text === "") {
        return null
    }
    const parsed = parse_kftl_date_time(text)
    if (parsed === null) {
        throw new Error(i18n.global.t("KFTL_REPEAT_INVALID_ORIGIN_MESSAGE_TITLE"))
    }
    return parsed
}

// ─── 必須2行の検査 ───────────────────────────────────────────────────────────

export function validate_repeat_spec(spec: RepeatSpec): void {
    if (spec.cond === null) {
        throw new Error(i18n.global.t("KFTL_REPEAT_CONDITION_REQUIRED_MESSAGE_TITLE"))
    }
    if (spec.count === 0 && spec.until === null) {
        throw new Error(i18n.global.t("KFTL_REPEAT_COUNT_REQUIRED_MESSAGE_TITLE"))
    }
}

// ─── 候補日時の計算 ───────────────────────────────────────────────────────────

/**
 * 条件に一致する日時を順に返す。
 *
 * 時刻はアンカー（そのレコードが持つ日時欄のうち最初に埋まっているもの）から取る。
 * **起点ちょうどは含めない**（`候補 > 起点`）。
 */
export function occurrences_of(spec: RepeatSpec, anchor: Date, base: Date): Array<Date> {
    if (spec.cond === null) {
        throw new Error(i18n.global.t("KFTL_REPEAT_CONDITION_REQUIRED_MESSAGE_TITLE"))
    }
    const cond = spec.cond
    const origin = spec.origin !== null ? spec.origin : base
    const at = (year: number, month: number, day: number): Date =>
        new Date(year, month, day, anchor.getHours(), anchor.getMinutes(), anchor.getSeconds())

    const out: Array<Date> = []
    // accept は候補を1つ受け取り、走査を続けてよいかを返す
    const accept = (candidate: Date): boolean => {
        if (candidate.getTime() <= origin.getTime()) {
            return true // 起点以前。まだ先に候補がある
        }
        if (spec.until !== null && candidate.getTime() > spec.until.getTime()) {
            return false
        }
        out.push(candidate)
        if (out.length > REPEAT_MAX_RECORDS) {
            throw new Error(i18n.global.t("KFTL_REPEAT_LIMIT_EXCEEDED_MESSAGE_TITLE"))
        }
        return !(spec.count > 0 && out.length >= spec.count)
    }

    if (cond.kind === 'nth_weekday' || cond.kind === 'month_day') {
        let year = origin.getFullYear()
        let month = origin.getMonth()
        for (let i = 0; i <= REPEAT_SCAN_MONTHS; i++) {
            const day = monthly_candidate(cond, year, month)
            if (day !== null && !accept(at(year, month, day))) {
                break
            }
            month++
            if (month > 11) {
                month = 0
                year++
            }
        }
    } else if (cond.kind === 'weekday' && cond.week_interval > 1) {
        // N週おきは「最初の一致」を基準にして、そこから N 週ずつ送る。
        // 日送りで拾うと基準週が決まらない
        const weekday = cond.weekdays[0]
        const cursor = new Date(origin.getFullYear(), origin.getMonth(), origin.getDate())
        let found = false
        for (let i = 0; i < 8; i++) { // 今日がその曜日でも時刻が過ぎていれば翌週まで送る
            if (cursor.getDay() === weekday && at(cursor.getFullYear(), cursor.getMonth(), cursor.getDate()).getTime() > origin.getTime()) {
                found = true
                break
            }
            cursor.setDate(cursor.getDate() + 1)
        }
        if (found) {
            const step = 7 * cond.week_interval
            for (let i = 0; i * step <= REPEAT_SCAN_DAYS; i++) {
                if (!accept(at(cursor.getFullYear(), cursor.getMonth(), cursor.getDate()))) {
                    break
                }
                cursor.setDate(cursor.getDate() + step)
            }
        }
    } else {
        const cursor = new Date(origin.getFullYear(), origin.getMonth(), origin.getDate())
        for (let i = 0; i <= REPEAT_SCAN_DAYS; i++) {
            if (matches_daily_or_weekday(cond, cursor) && !accept(at(cursor.getFullYear(), cursor.getMonth(), cursor.getDate()))) {
                break
            }
            cursor.setDate(cursor.getDate() + 1)
        }
    }
    return out
}

function matches_daily_or_weekday(cond: RepeatCond, date: Date): boolean {
    if (cond.kind === 'daily') {
        return true
    }
    return cond.weekdays.includes(date.getDay())
}

/**
 * その月の候補日（日にちだけ）を返す。無ければ null。
 *
 * **存在しない日はその月を飛ばす**（第5金が無い月、2月の「毎月31」など）。
 * 最も近い日へ丸めると「第4金」「毎月30」と重複するので、丸めない。
 */
function monthly_candidate(cond: RepeatCond, year: number, month: number): number | null {
    if (cond.kind === 'month_day') {
        const date = new Date(year, month, cond.month_day)
        return date.getMonth() !== month ? null : cond.month_day
    }
    if (cond.kind === 'nth_weekday') {
        const weekday = cond.weekdays[0]
        if (cond.nth === -1) {
            // 「翌月の0日」= 当月の末日
            const last = new Date(year, month + 1, 0)
            while (last.getDay() !== weekday) {
                last.setDate(last.getDate() - 1)
            }
            return last.getDate()
        }
        const first = new Date(year, month, 1)
        const offset = (weekday - first.getDay() + 7) % 7
        const date = new Date(year, month, 1 + offset + (cond.nth - 1) * 7)
        return date.getMonth() !== month ? null : date.getDate()
    }
    return null
}

// ─── 日数のずらし ─────────────────────────────────────────────────────────────

/**
 * 暦日数でずらす。ミリ秒加算にしないのは壁時計時刻を保つため
 * （夏時間のある地域で時刻がずれる）。
 */
export function shift_days(date: Date, day_shift: number): Date {
    if (day_shift === 0) {
        return new Date(date.getTime())
    }
    const shifted = new Date(date.getTime())
    shifted.setDate(shifted.getDate() + day_shift)
    return shifted
}

/** 暦日の差。時刻を落としてから引くので夏時間の影響を受けない。 */
export function days_between(from: Date, to: Date): number {
    const a = Date.UTC(from.getFullYear(), from.getMonth(), from.getDate())
    const b = Date.UTC(to.getFullYear(), to.getMonth(), to.getDate())
    return Math.round((b - a) / 86400000)
}
