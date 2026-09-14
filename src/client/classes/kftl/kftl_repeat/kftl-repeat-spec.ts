'use strict'

import { i18n } from '@/i18n'
import { parse_kftl_date_time } from '../kftl-date-time'

/**
 * 繰り返しブロック「？？」の4行の読み方。**行ラベル（「毎週金曜」「3回」の表示）のためだけ**にある。
 *
 * 候補日時の計算・展開・既存判定・上限はサーバの Go 実装（kftl_repeat.go / kftl_repeat_lines.go）
 * だけが持つ（ADR-0507）。ここに展開を戻さないこと ―― `use-kftl-view.ts` は本文が変わるたびに
 * 全行を分類し直すので、打鍵1回あたり最大1000件を作ることになる。
 *
 * Mirrors: src/server/gkill/api/kftl/kftl_repeat.go（parse 系のみ）
 */

/** 1つの「？？」ブロック、および1回の送信が作れるレコード数の上限（Go の repeatMaxRecords と同じ値）。回数行のラベルの判定に使う。 */
export const REPEAT_MAX_RECORDS = 1000

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
