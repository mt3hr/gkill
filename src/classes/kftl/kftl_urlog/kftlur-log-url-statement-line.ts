'use strict'

import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLURLogRequest } from './kftlur-log-request'
import { KFTLURLogTitleStatementLine } from './kftlur-log-title-statement-line'

export class KFTLURLogURLStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLURLogTitleStatementLine(line_text, context))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLURLogRequest
        req.set_url(this.get_statement_line_text())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return "URL"
    }

}


