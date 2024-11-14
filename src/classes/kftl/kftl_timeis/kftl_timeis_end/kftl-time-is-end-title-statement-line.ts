'use strict'

import type { KFTLRequestMap } from '../../kftl-request-map'
import { KFTLStatementLine } from '../../kftl-statement-line'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { KFTLTimeIsEndByTitleRequest } from './kftl-time-is-end-by-title-request'

export class KFTLTimeIsEndTitleStatementLine extends KFTLStatementLine {
    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = new KFTLTimeIsEndByTitleRequest(this.get_context().get_this_statement_line_target_id(), this.get_context())
        req.set_title(this.get_context().get_this_statement_line_text())
        req.set_error_when_target_does_not_exist(true)
        request_map.set(this.get_context().get_this_statement_line_target_id(), req)
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "timeis"
    }

    static is_this_type(line_text: string): boolean {
        return line_text == "ーえ"
    }
}


