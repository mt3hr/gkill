'use strict'

import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLMiLimitTimeStatementLine } from './kftl-mi-limit-time-statement-line'
import type { KFTLMiRequest } from './kftl-mi-request'

export class KFTLMiBoardNameStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLMiLimitTimeStatementLine(line_text, context))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const mi_request = request_map.get(this.get_context().get_this_statement_line_target_id()) as unknown as KFTLMiRequest
        mi_request.set_board_name(this.get_statement_line_text())
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return "板名"
    }

}