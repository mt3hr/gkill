'use strict'

import moment from 'moment'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLPrototypeRequest } from '../kftl_prototype/kftl-prototype-request'
import type { KFTLTimeIsRequest } from './kftl-time-is-request'
import { KFTLTimeIsEndTimeStatementLine } from './kftl-time-is-end-time-statement-line'

export class KFTLTimeIsStartTimeStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLTimeIsEndTimeStatementLine(line_text, context))
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
        request.set_related_time(time)
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "開始日時"
    }
}


