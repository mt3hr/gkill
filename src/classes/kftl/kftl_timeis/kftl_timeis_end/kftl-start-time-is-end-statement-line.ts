'use strict'

import moment from 'moment'
import type { KFTLRequestMap } from '../../kftl-request-map'
import { KFTLStatementLine } from '../../kftl-statement-line'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { KFTLPrototypeRequest } from '../../kftl_prototype/kftl-prototype-request'
import type { KFTLTimeIsRequest } from '../kftl-time-is-request'
import { GkillAPI } from '@/classes/api/gkill-api'

export class KFTLStartTimeIsEndStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        const target_id = this.get_prev_line() && this.get_prev_line()?.get_context() && this.get_prev_line()?.get_context().is_this_prototype() ? this.get_prev_line()!!.get_context().get_this_statement_line_target_id() : GkillAPI.get_gkill_api().generate_uuid()
        context.set_this_statement_line_target_id(target_id)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLStartTimeIsEndStatementLine(line_text, context))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        let request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLTimeIsRequest
        if (!request) {
            request_map.set(this.get_context().get_this_statement_line_target_id(), new KFTLPrototypeRequest(this.get_context().get_this_statement_line_target_id(), this.get_context()))
            request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLTimeIsRequest
        }
        const time = moment(this.get_context().get_this_statement_line_text().replace("？", "")).toDate()
        if (Number.isNaN(time.getTime())) {
            throw new Error("日時の解釈に失敗しました")
        }
        request.set_end_time(time)
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "終了日時"
    }

    static is_this_type(line_text: string): boolean {
        return line_text == "ーえ"
    }

}


