'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLKCNumValueStatementLine } from './kftl-kc-num-value-statement-line'
import type { KFTLKCRequest } from './kftl-kc-request'

export class KFTLKCTitleStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor((line_text, context) => new KFTLKCNumValueStatementLine(line_text, context))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLKCRequest
        req.set_title(this.get_context().get_this_statement_line_text())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_KC_TITLE_TITLE")
    }

}


