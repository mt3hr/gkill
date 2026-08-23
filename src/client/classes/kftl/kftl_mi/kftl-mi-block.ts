'use strict'

import type { KFTLBlockReentryProvider, KFTLStatementLineConstructor } from '../kftl-statement-line'
import { KFTLTagStatementLine } from '../kftl_tag/kftl-tag-statement-line'
import { KFTLStartTextStatementLine } from '../kftl_text/kftl-start-text-statement-line'

/**
 * タスクブロック(`ーみ` / `/mi`)の中の「次の行」を決める先読み。
 *
 * タグ行・テキスト開始行は**項目の位置を消費しない**。汎用の行クラスをそのまま使い、
 * 「ブロックへ復帰する次行の決め方」だけを渡す(支出ブロックと同じやり方)。
 * 渡さないと、ブロックの途中に `。タグ` と書いた時点でその行が板名や見積開始として
 * 読まれてしまい、タグは付かないまま板名が "。タグ" になる。
 *
 * タスクのリクエストは this_statement_line_target_id をキーに request_map へ入っている
 * (KFTLStartMiStatementLine.apply_this_line_to_request_map)ので、
 * リポストタスクと違って専用のタグ行は要らない。汎用のタグ行がそのままタスクへタグを付ける。
 *
 * **`？` はここで拾ってはいけない。** 見積開始・見積終了・期限の3行は
 * `？`/`?` を任意の接頭辞として自分で剥がす。関連時刻行へ回すと
 * 空行で位置を送る既存の書き方が壊れる(タスクに関連時刻の項目は無い)。
 *
 * 空行も拾わない。空行は今までどおり項目の位置を消費する。
 * Mirrors: generateMiBlockNextConstructor (kftl_mi.go)
 */
export function generate_mi_block_next_constructor(next_line_text: string, next_field_constructor: KFTLStatementLineConstructor): KFTLStatementLineConstructor {
    const reentry: KFTLBlockReentryProvider = (line_text: string) => generate_mi_block_next_constructor(line_text, next_field_constructor)

    if (KFTLTagStatementLine.is_this_type(next_line_text)) {
        return (line_text: string, context) => new KFTLTagStatementLine(line_text, context, false, reentry)
    }
    if (KFTLStartTextStatementLine.is_this_type(next_line_text)) {
        return (line_text: string, context) => new KFTLStartTextStatementLine(line_text, context, false, reentry)
    }
    return next_field_constructor
}
