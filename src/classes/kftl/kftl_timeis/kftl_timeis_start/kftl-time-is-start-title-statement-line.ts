'use strict'

import type { KFTLRequestMap } from '../../kftl-request-map'
import { KFTLStatementLine } from '../../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import type { KFTLTimeIsStartRequest } from './kftl-time-is-start-request'

export class KFTLTimeIsStartTitleStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(this.get_context().get_next_statement_line_text()))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLTimeIsStartRequest
        req.set_title(this.get_context().get_this_statement_line_text())
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "開始"
    }

}


