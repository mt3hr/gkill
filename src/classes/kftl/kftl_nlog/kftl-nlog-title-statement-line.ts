'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLNlogAmountStatementLine } from './kftl-nlog-amount-statement-line'
import type { KFTLNlogRequest } from './kftl-nlog-request'

export class KFTLNlogTitleStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLNlogAmountStatementLine(line_text, context))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const nlog_request = request_map.get(this.get_context().get_this_statement_line_target_id()) as unknown as KFTLNlogRequest
        nlog_request.add_title(this.get_context().get_this_statement_line_text())
        return new Promise<void>((resolve) => resolve())

    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_NLOG_TITLE_TITLE")
    }

}


