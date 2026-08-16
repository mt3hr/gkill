'use strict'

import { i18n } from '@/i18n'
import { GkillAPI } from '@/classes/api/gkill-api'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLNlogAmountStatementLine } from './kftl-nlog-amount-statement-line'
import { KFTLNlogRequest } from './kftl-nlog-request'
import { assert_is_not_meta_info_line, type KFTLNlogBlock } from './kftl-nlog-block'

/**
 * 支出ブロックの品名行。ここが1件の支払いの始まりになる。
 *
 * 支払いごとに target_id を採番し直すので、金額行のあとに書いたタグ・テキストは
 * 汎用のタグ行・テキスト行のまま「その支払い」に付く
 */
export class KFTLNlogTitleStatementLine extends KFTLStatementLine {

    private block: KFTLNlogBlock

    constructor(line_text: string, context: KFTLStatementLineContext, block: KFTLNlogBlock) {
        super(line_text, context)
        this.block = block
        const payment_id = GkillAPI.get_gkill_api().generate_uuid()
        context.set_this_statement_line_target_id(payment_id)
        context.set_next_statement_line_target_id(payment_id)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLNlogAmountStatementLine(line_text, context, block))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        // 店名の次は品名行で固定なので、ここへタグ行やテキスト開始行が来ると
        // その文字列が品名になってしまう。飲み込まずにおかしな行として出す
        assert_is_not_meta_info_line(this.get_context().get_this_statement_line_text())
        const nlog_request = new KFTLNlogRequest(this.get_context().get_this_statement_line_target_id(), this.get_context(), this.block)
        nlog_request.set_title(this.get_context().get_this_statement_line_text())
        request_map.set(this.get_context().get_this_statement_line_target_id(), nlog_request)
        return new Promise<void>((resolve) => resolve())

    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_NLOG_TITLE_TITLE")
    }

}
