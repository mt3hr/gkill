'use strict'

import { GkillAPI } from '@/classes/api/gkill-api'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLNlogShopNameStatementLine } from './kftl-nlog-shop-name-statement-line'
import { KFTLNlogBlock } from './kftl-nlog-block'
import { KFTLPrototypeRequest } from '../kftl_prototype/kftl-prototype-request'
import { i18n } from '@/i18n'
import { KFTL_ASCII_NLOG_SPLITTER_TITLE, matches_exact } from '../kftl-prefixes'

export class KFTLStartNlogStatementLine extends KFTLStatementLine {

    private block: KFTLNlogBlock

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        const target_id = this.get_prev_line() && this.get_prev_line()?.get_context() && this.get_prev_line()?.get_context().is_this_prototype() ? this.get_prev_line()!.get_context().get_this_statement_line_target_id() : GkillAPI.get_gkill_api().generate_uuid()
        this.block = new KFTLNlogBlock(target_id)
        context.set_this_statement_line_target_id(target_id)
        context.set_next_statement_line_target_id(target_id)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLNlogShopNameStatementLine(line_text, context, this.block))
    }

    /**
     * 開始行ではリクエストを作らない。支払いは品名行が1組ずつ作る。
     *
     * ここでやるのは「`ーん` より前に書かれたメタ情報行」の検査だけ。
     * 支出のタグとテキストは直前の支払いに付ける仕様なので、ブロックの前に書かれていたら
     * 黙って捨てずにおかしな行として出す(以前はブロックに1回だけ付き、しかもクライアントでは
     * どの Nlog にも紐づかずに消えていた)。関連時刻だけはブロック全体に効くので取り込む。
     */
    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const prev_request = request_map.get(this.get_context().get_this_statement_line_target_id())
        if (prev_request) {
            if (!KFTLPrototypeRequest.is_prototype_request(prev_request)) {
                // 区切らずにメモの直後へ書いた場合。従来 KFTLRequestMap.set が投げていたのと同じエラー
                throw new Error(i18n.global.t('KFTL_REQUEST_ALREADY_SET_ERROR_MESSAGE', [this.get_context().get_this_statement_line_target_id()]))
            }
            if (0 < prev_request.get_tags().length || 0 < prev_request.get_texts().length) {
                throw new Error(i18n.global.t('KFTL_NLOG_META_INFO_MUST_BE_AFTER_AMOUNT_MESSAGE_TITLE'))
            }
            this.block.related_time = prev_request.get_raw_related_time()
        }
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_NLOG_NLOG_LABEL_TITLE")
    }

    static is_this_type(line_text: string): boolean {
        return matches_exact(line_text, "KFTL_NLOG_SPLITTER_TITLE", KFTL_ASCII_NLOG_SPLITTER_TITLE)
    }

}
