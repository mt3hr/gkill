'use strict'

import { GkillAPI } from '@/classes/api/gkill-api'
import type { KFTLRequestMap } from '../../kftl-request-map'
import { KFTLStatementLine } from '../../kftl-statement-line'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { KFTLTimeIsEndByTitleRequest } from './kftl-time-is-end-by-title-request'
import { KFTLStatementLineConstructorFactory } from '../../kftl-statement-line-constructor-factory'

export class KFTLTimeIsEndTitleStatementLine extends KFTLStatementLine {
    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(this.get_context().get_next_statement_line_text()))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = new KFTLTimeIsEndByTitleRequest(this.get_context().get_this_statement_line_target_id(), this.get_context())
        req.set_title(this.get_context().get_this_statement_line_text())
        req.set_error_when_target_does_not_exist(true)
        request_map.set(this.get_context().get_this_statement_line_target_id(), req)
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "終了"
    }
}


