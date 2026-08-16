'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLMiReKyouRequest } from './kftl-mi-re-kyou-request'
import { KFTLEndMiReKyouStatementLine } from './kftl-end-mi-re-kyou-statement-line'
import { KFTL_ASCII_TAG_PREFIX, matches_prefix, split_tags, strip_prefix } from '../kftl-prefixes'

// ブロックの次の行を組み立てる関数。項目行(板名・見積開始・見積終了・期日)ごとに中身が違う
export type KFTLMiReKyouNextLineConstructor = { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine }

/**
 * リポストタスクのブロックの中のタグ行。
 *
 * ここで足したタグは対象のKyouではなくMiReKyou自身に付く。
 * target_idを切り替えるのではなく、掴んでいるリクエストへ直接add_tagする
 * (target_idは対象のKyouのままブロックを通り抜けるので、閉じたあとのタグは対象に付く)。
 *
 * タグ行は項目行の位置を消費しないので、板名の前でも後でも、項目行の合間でも書ける。
 */
export class KFTLMiReKyouTagStatementLine extends KFTLStatementLine {

    private request: KFTLMiReKyouRequest

    constructor(line_text: string, context: KFTLStatementLineContext, request: KFTLMiReKyouRequest, prev_line_is_meta_info: boolean, next_field_constructor: KFTLMiReKyouNextLineConstructor) {
        super(line_text, context)
        this.request = request
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLMiReKyouTagStatementLine.generate_next_constructor(context.get_next_statement_line_text(), request, prev_line_is_meta_info, next_field_constructor))
    }

    async apply_this_line_to_request_map(_request_map: KFTLRequestMap): Promise<void> {
        const line_text = this.get_context().get_this_statement_line_text()
        if (line_text == "" || line_text == "\n") {
            return new Promise<void>((resolve) => resolve())
        }
        if (!matches_prefix(line_text, "KFTL_TAG_PREFIX", KFTL_ASCII_TAG_PREFIX)) {
            // 項目行を全部書き終えたあとに来られるのはタグ行か閉じる行だけ。
            // テキストは飲み込まずにおかしな行として出す。飲み込むと、閉じ忘れたときに
            // メモの本文が丸ごとタグになってしまう
            throw new Error(i18n.global.t('KFTL_NONE_VALUE_IS_NOT_BLANK_MESSAGE_TITLE'))
        }
        const tags: Array<string> = split_tags(strip_prefix(line_text, "KFTL_TAG_PREFIX", KFTL_ASCII_TAG_PREFIX))
        tags.forEach(tag => {
            this.request.add_tag(tag)
        })
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TAG_LABEL_TITLE")
    }

    /**
     * ブロックの中の「次の行」を決める先読み。
     *
     * 閉じる行 > タグ行 > 次の項目行 の順に見る。
     * タグ行は項目の位置を消費しないので、タグを挟んでも次の非タグ行が次の項目になる。
     */
    static generate_next_constructor(next_line_text: string, request: KFTLMiReKyouRequest, prev_line_is_meta_info: boolean, next_field_constructor: KFTLMiReKyouNextLineConstructor): KFTLMiReKyouNextLineConstructor {
        if (KFTLEndMiReKyouStatementLine.is_this_type(next_line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => new KFTLEndMiReKyouStatementLine(line_text, context, prev_line_is_meta_info)
        }
        if (matches_prefix(next_line_text, "KFTL_TAG_PREFIX", KFTL_ASCII_TAG_PREFIX)) {
            return (line_text: string, context: KFTLStatementLineContext) => new KFTLMiReKyouTagStatementLine(line_text, context, request, prev_line_is_meta_info, next_field_constructor)
        }
        return next_field_constructor
    }

    /**
     * 項目行を書き終えたあとの「次の行」。タグ行か閉じる行しか来られない。
     * それ以外の行はタグ行として作られてapply時におかしな行になる
     */
    static generate_after_last_field_constructor(next_line_text: string, request: KFTLMiReKyouRequest, prev_line_is_meta_info: boolean): KFTLMiReKyouNextLineConstructor {
        const stay_in_block: KFTLMiReKyouNextLineConstructor = (line_text: string, context: KFTLStatementLineContext) => new KFTLMiReKyouTagStatementLine(line_text, context, request, prev_line_is_meta_info, stay_in_block)
        return KFTLMiReKyouTagStatementLine.generate_next_constructor(next_line_text, request, prev_line_is_meta_info, stay_in_block)
    }
}
