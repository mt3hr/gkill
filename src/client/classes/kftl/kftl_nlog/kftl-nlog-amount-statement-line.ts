'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLNlogRequest } from './kftl-nlog-request'
import { generate_nlog_block_next_constructor, type KFTLNlogBlock } from './kftl-nlog-block'

export class KFTLNlogAmountStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, block: KFTLNlogBlock) {
        super(line_text, context)
        // target_id は支払いのまま。次の行がタグ行やテキストブロックなら、
        // その支払いに付いたうえでブロックの中に留まる
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(generate_nlog_block_next_constructor(this.get_context().get_next_statement_line_text(), block))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const amount = this.parse_amount()
        const nlog_request = request_map.get(this.get_context().get_this_statement_line_target_id()) as unknown as KFTLNlogRequest
        nlog_request.set_amount(amount)
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        try {
            const line_text = this.get_context().get_this_statement_line_text()
            if (line_text == "" || line_text == "\n") {
                return i18n.global.t("KFTL_NLOG_AMOUNT_LABEL_TITLE")
            }
            const amount = this.parse_amount()
            if (0 < amount) {
                return i18n.global.t("KFTL_NLOG_IN_LABEL_TITLE")
            } else {
                return i18n.global.t("KFTL_NLOG_OUT_LABEL_TITLE")
            }
        } catch (_e: unknown) {
            return i18n.global.t("KFTL_NLOG_INVALID_AMOUNT_LABEL_TITLE")
        }
    }

    private parse_amount(): number {
        try {
            // 小数を落とさないこと。parseInt だと 1.5 が黙って 1 になり、
            // サーバ側パーサ(json.Number)とも食い違う
            const amount = Number.parseFloat(this.get_context().get_this_statement_line_text().trim())
            if (Number.isNaN(amount)) {
                throw new Error(i18n.global.t("KFTL_NLOG_INVALID_AMOUNT_MESSAGE_TITLE"))
            }
            return amount
        } catch (_e: unknown) {
            throw new Error(i18n.global.t("KFTL_NLOG_INVALID_AMOUNT_MESSAGE_TITLE"))
        }
    }

}
