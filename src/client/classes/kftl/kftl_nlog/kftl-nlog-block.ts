'use strict'

import { i18n } from '@/i18n'
import type { KFTLBlockReentryProvider, KFTLStatementLineConstructor } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import { KFTLTagStatementLine } from '../kftl_tag/kftl-tag-statement-line'
import { KFTLStartTextStatementLine } from '../kftl_text/kftl-start-text-statement-line'
import { KFTLRelatedTimeStatementLine } from '../kftl_related_time/kftl-related-time-statement-line'
import { KFTLNlogRelatedTimeStatementLine } from './kftl-nlog-related-time-statement-line'

/**
 * 支出ブロック(`ーん` / `/expense`)で、支払いをまたいで共有する状態。
 *
 * 支払い(品名と金額のペア)は1組ずつ別の KFTLNlogRequest になるので、
 * タグとテキストは「直前の支払い」だけに付く。全支払いで共有するのは
 * 店名と関連時刻の2つだけで、それをここに置く。
 *
 * 開始行が1つ作って、ブロックの中の行へクロージャで渡していく
 * (kftl_mirekyou がリクエストを持ち回るのと同じやり方)。
 */
export class KFTLNlogBlock {

    // 全支払いで共有する店名
    shop_name: string

    // ブロックの入口の target_id。`ーん` の前に書かれたメタ情報行を見つけるために持つ
    block_target_id: string

    // `？`行で指定された関連時刻。ブロックの中のどこに書いてもブロック全体に効く
    related_time: Date | null

    constructor(block_target_id: string) {
        this.shop_name = ""
        this.block_target_id = block_target_id
        this.related_time = null
    }
}

/**
 * 支出ブロックの中の「次の行」を決める先読み。
 *
 * タグ行・テキスト開始行は汎用の行クラスをそのまま使い、「ブロックへ復帰する次行の決め方」
 * だけを渡す。汎用の行クラスは既定では次を kmemo か none にするので、渡さないと
 * ブロックの途中にタグを書いた時点でブロックが切れて、以降の品名行が拾われなくなる。
 *
 * 関連時刻だけは付け先が支払いではなくブロックなので専用の行クラスにする。
 *
 * どれでもなければ従来どおり generate_nlog_constructor へ委譲する
 * (`、` や `ーみ` などはそちらが拾い、何にも当たらなければ次の品名行になる)。
 */
export function generate_nlog_block_next_constructor(next_line_text: string, block: KFTLNlogBlock): KFTLStatementLineConstructor {
    const reentry: KFTLBlockReentryProvider = (line_text: string) => generate_nlog_block_next_constructor(line_text, block)

    if (KFTLTagStatementLine.is_this_type(next_line_text)) {
        return (line_text: string, context) => new KFTLTagStatementLine(line_text, context, false, reentry)
    }
    if (KFTLStartTextStatementLine.is_this_type(next_line_text)) {
        return (line_text: string, context) => new KFTLStartTextStatementLine(line_text, context, false, reentry)
    }
    if (KFTLRelatedTimeStatementLine.is_this_type(next_line_text)) {
        return (line_text: string, context) => new KFTLNlogRelatedTimeStatementLine(line_text, context, block)
    }
    return KFTLStatementLineConstructorFactory.get_instance().generate_nlog_constructor(next_line_text, block)
}

/**
 * 店名行・最初の品名行が、タグ行やテキストブロックの開始行になっていないか検査する。
 *
 * この2箇所は次の行が固定なので、`。タグ` と書くと店名や品名がその文字列になってしまう。
 * 支出のタグは「直前の支払い」に付ける仕様で、この位置には直前の支払いがまだ無いので、
 * 黙って飲み込まずにおかしな行として出す
 */
export function assert_is_not_meta_info_line(line_text: string): void {
    if (KFTLTagStatementLine.is_this_type(line_text) || KFTLStartTextStatementLine.is_this_type(line_text)) {
        throw new Error(i18n.global.t('KFTL_NLOG_META_INFO_MUST_BE_AFTER_AMOUNT_MESSAGE_TITLE'))
    }
}
