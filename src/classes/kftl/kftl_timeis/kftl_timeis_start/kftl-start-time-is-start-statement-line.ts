'use strict'

import { GkillAPI } from '@/classes/api/gkill-api'
import type { KFTLRequestMap } from '../../kftl-request-map'
import { KFTLStatementLine } from '../../kftl-statement-line'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { KFTLTimeIsStartRequest } from './kftl-time-is-start-request'
import { KFTLTimeIsStartTitleStatementLine } from './kftl-time-is-start-title-statement-line'

export class KFTLStartTimeIsStartStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        const target_id = this.get_prev_line() && this.get_prev_line()?.get_context() && this.get_prev_line()?.get_context().is_this_prototype() ? this.get_prev_line()!!.get_context().get_this_statement_line_target_id() : GkillAPI.get_gkill_api().generate_uuid()
        context.set_this_statement_line_target_id(target_id)
        context.set_next_statement_line_target_id(target_id)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLTimeIsStartTitleStatementLine(line_text, context))

    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = new KFTLTimeIsStartRequest(this.get_context().get_this_statement_line_target_id(), this.get_context())
        request_map.set(this.get_context().get_this_statement_line_target_id(), req)
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "timeis"
    }

    static is_this_type(line_text: string): boolean {
        return line_text == "ーた"
    }

}


