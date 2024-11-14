'use strict'

import type { KFTLRequestMap } from '@/classes/kftl/kftl-request-map'
import type { KFTLStatementLineContext } from '@/classes/kftl/kftl-statement-line-context'
import { KFTLStatementLine } from '../../../kftl-statement-line'
import { KFTLTimeIsEndByTagRequest } from '../kftl-time-is-end-by-tag-request'
import { GkillAPI } from '@/classes/api/gkill-api'
import { KFTLTimeIsEndByTagTagNameStatementLine } from './kftl-time-is-end-by-tag-tag-name-statement-line'

export class KFTLStartTimeIsEndByTagStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        const target_id = context.is_this_prototype() && this.get_prev_line() ? this.get_prev_line()!!.get_context().get_this_statement_line_target_id() : GkillAPI.get_instance().generate_uuid()
        context.set_this_statement_line_target_id(target_id)
        context.set_next_statement_line_target_id(target_id)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLTimeIsEndByTagTagNameStatementLine(line_text, context))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = new KFTLTimeIsEndByTagRequest(this.get_context().get_this_statement_line_target_id(), this.get_context())
        req.set_error_when_target_does_not_exist(true)
        request_map.set(this.get_context().get_this_statement_line_target_id(), req)
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "timeis"
    }

    static is_this_type(line_text: string): boolean {
        return line_text == "ーたえ"
    }
}