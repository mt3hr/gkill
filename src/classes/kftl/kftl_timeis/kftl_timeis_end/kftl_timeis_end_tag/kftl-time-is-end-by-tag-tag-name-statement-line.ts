'use strict'

import type { KFTLRequestMap } from '@/classes/kftl/kftl-request-map'
import type { KFTLStatementLineContext } from '@/classes/kftl/kftl-statement-line-context'
import { KFTLStatementLine } from '../../../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '@/classes/kftl/kftl-statement-line-constructor-factory'
import type { KFTLTimeIsEndByTagRequest } from '../kftl-time-is-end-by-tag-request'

export class KFTLTimeIsEndByTagTagNameStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(this.get_context().get_next_statement_line_text()))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLTimeIsEndByTagRequest
        for (let i = 0; i < this.get_context().get_this_statement_line_text().split("、").length; i++) {
            const tag = this.get_context().get_this_statement_line_text().split("、")[i].trim()
            if (tag == "") { continue }
            req.add_target_tag_name(tag)
        }
        req.set_error_when_target_does_not_exist(true)
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "終了"
    }

}


