'use strict'

import moment from 'moment'

// KFTL の関連時刻・打刻時刻の書式。Go 側 kftl_related_time_statement_line.go の
// dateFormats と対。moment に書式を明示して strict パースすると、欠けた年月日は
// 「今日」で補完される（時刻のみ→当日、月日のみ→現在年）ので Go の修正後挙動と一致する。
// これをやらないと素の moment(text) が new Date フォールバックに落ち、
//   - 時刻のみ「15:04」→ Invalid Date（拒否）
//   - 月日のみ「1/2 15:04」→ Chromium は既定年2001で黙って過去日付
// というブラウザ依存の壊れ方をする。
const KFTL_DATE_TIME_FORMATS = [
    'YYYY-MM-DD HH:mm:ss',
    'YYYY-MM-DDTHH:mm:ss',
    'YYYY/MM/DD HH:mm:ss',
    'YYYY/MM/DDTHH:mm:ss',
    'YYYY-MM-DD HH:mm',
    'YYYY/MM/DD HH:mm',
    'YYYY-MM-DD',
    'YYYY/MM/DD',
    'MM/DD HH:mm',
    'M/D HH:mm',
    'HH:mm:ss',
    'H:mm:ss',
    'HH:mm',
    'H:mm',
]

// parse_kftl_date_time は KFTL の時刻文字列を Date に変換する。解釈できなければ null。
// まず書式を明示した strict パースを試し（欠けた年月日は当日/現在年で補完）、
// 全書式に一致しないときだけ従来の裸 moment(text) へフォールバックして
// 受理集合を狭めない（ISOミリ秒付き等の後方互換）。
export function parse_kftl_date_time(text: string): Date | null {
    const strict = moment(text, KFTL_DATE_TIME_FORMATS, true)
    if (strict.isValid()) {
        return strict.toDate()
    }
    const loose = moment(text)
    if (loose.isValid()) {
        return loose.toDate()
    }
    return null
}
