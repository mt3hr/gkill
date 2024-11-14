'use strict'

import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLTimeIsRequest } from './kftl-time-is-request'
import { KFTLTimeIsStartTimeStatementLine } from './kftl-time-is-start-time-statement-line'

export class KFTLTimeIsTitleStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLTimeIsStartTimeStatementLine(line_text, context))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLTimeIsRequest
        req.set_title(this.get_context().get_this_statement_line_text())
        return new Promise<void>((resolve) => resolve())

    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "タイトル"
    }

}


