'use strict'

import type { KFTLRequestMap } from '@/classes/kftl/kftl-request-map'
import type { KFTLStatementLineContext } from '@/classes/kftl/kftl-statement-line-context'
import { KFTLStatementLine } from '../../../kftl-statement-line'
import { GkillAPI } from '@/classes/api/gkill-api'
import { KFTLTimeIsEndByTagIfExistTagNameStatementLine } from './kftl-time-is-end-by-tag-if-exist-tag-name-statement-line'

export class KFTLStartTimeIsEndByTagIfExistStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        const target_id = context.is_this_prototype() && this.get_prev_line() ? this.get_prev_line()!!.get_context().get_this_statement_line_target_id() : GkillAPI.get_gkill_api().generate_uuid()
        context.set_this_statement_line_target_id(target_id)
        context.set_next_statement_line_target_id(target_id)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLTimeIsEndByTagIfExistTagNameStatementLine(line_text, context))
    }

    async apply_this_line_to_request_map(requet_map: KFTLRequestMap): Promise<void> {
        throw new Error('Not implemented')
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "timeis"
    }

    static is_this_type(line_text: string): boolean {
        return line_text == "ーいたえ"
    }

}


