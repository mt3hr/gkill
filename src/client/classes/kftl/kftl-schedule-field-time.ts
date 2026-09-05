'use strict'

import { i18n } from '@/i18n'
import { parse_kftl_date_time } from './kftl-date-time'
import { KFTL_ASCII_TIMEIS_TIME_PREFIX, matches_prefix } from './kftl-prefixes'

/**
 * Mi / MiReKyou の予定日時欄（見積開始・見積終了・期限）の1行を読む。
 *
 * 行頭の「？」「?」は**例外を投げる**。行の解釈が例外を投げると、その行は
 * `get_invalid_line_indexs` に拾われてピンクになり、保存も止まる。
 *
 * 以前は関連時刻と同じ接頭辞として黙って剥がしていたが、剥がしたあとにパースへ
 * 失敗しても null を返して未設定として握り潰す作りなので、「？18:00」の打ち間違いも
 * 「？？」（繰り返しブロック）の書き損じも、エラーも警告も出ないまま日付だけが
 * 入らない形で落ちていた。剥がす前に弾く。Mi は related_time を持たないので、
 * この欄に関連時刻の接頭辞を書けること自体に意味が無い。
 *
 * 空行は未設定（null）。それ以外の読めない行も今までどおり未設定として扱う
 * （日時として読めない行を一律に行エラーへ倒すと既存の書き方が広範に壊れるため、
 * そこは変えない）。
 *
 * Mirrors: parseScheduleFieldTime (src/server/gkill/api/kftl/kftl_related_time_statement_line.go)
 */
export function parse_schedule_field_time(line_text: string): Date | null {
    if (matches_prefix(line_text, "KFTL_TIMEIS_TIME_PREFIX", KFTL_ASCII_TIMEIS_TIME_PREFIX)) {
        throw new Error(i18n.global.t("KFTL_SCHEDULE_TIME_PREFIX_NOT_ALLOWED_MESSAGE_TITLE"))
    }
    if (line_text.trim() === "") {
        return null
    }
    return parse_kftl_date_time(line_text)
}
