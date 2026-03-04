'use strict'

import moment from 'moment'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLMiRequest } from './kftl-mi-request'
import { KFTLMiEstimateEndTimeStatementLine } from './kftl-mi-estimate-end-time-statement-line'
import { i18n } from '@/i18n'

export class KFTLMiEstimateStartTimeStatementLine extends KFTLStatementLine {
    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLMiEstimateEndTimeStatementLine(line_text, context))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())

    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const request = request_map.get(this.get_context().get_this_statement_line_target_id()) as unknown as KFTLMiRequest
        const time = moment(this.get_context().get_this_statement_line_text().replace("？", "")).toDate()
        if (!Number.isNaN(time.getTime())) {
            request.set_estimate_start_time(time)
        }
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        const line_text = this.get_context().get_this_statement_line_text()
        if (line_text == "" || line_text == "\n") {
            return i18n.global.t("KFTL_MI_NO_ESTIMATE_START_TIME_TITLE")
        }
        const time = moment(this.get_context().get_this_statement_line_text()).toDate()
        if (Number.isNaN(time.getTime())) {
            return i18n.global.t("KFTL_MI_INVALID_ESTIMATE_START_TIME_TITLE")
        }
        return i18n.global.t("KFTL_MI_ESTIMATE_START_TIME_TITLE")
    }
}


