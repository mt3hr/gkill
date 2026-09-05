'use strict'

import { parse_schedule_field_time } from '../kftl-schedule-field-time'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLMiRequest } from './kftl-mi-request'
import { generate_mi_block_next_constructor } from './kftl-mi-block'
import { KFTLMiLimitTimeStatementLine } from './kftl-mi-limit-time-statement-line'
import { i18n } from '@/i18n'

export class KFTLMiEstimateEndTimeStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_constructor(generate_mi_block_next_constructor(context.get_next_statement_line_text(), (line_text: string, context: KFTLStatementLineContext) => new KFTLMiLimitTimeStatementLine(line_text, context)))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const request = request_map.get(this.get_context().get_this_statement_line_target_id()) as unknown as KFTLMiRequest
        const time = parse_schedule_field_time(this.get_context().get_this_statement_line_text())
        if (time !== null) {
            request.set_estimate_end_time(time)
        }

    }

    get_label_name(_context: KFTLStatementLineContext): string {
        const line_text = this.get_context().get_this_statement_line_text()
        if (line_text == "" || line_text == "\n") {
            return i18n.global.t("KFTL_MI_NO_ESTIMATE_END_TIME_TITLE")
        }
        try {
            if (parse_schedule_field_time(line_text) === null) {
                return i18n.global.t("KFTL_MI_INVALID_ESTIMATE_END_TIME_TITLE")
            }
        } catch (_e: unknown) {
            return i18n.global.t("KFTL_MI_INVALID_ESTIMATE_END_TIME_TITLE")
        }
        return i18n.global.t("KFTL_MI_ESTIMATE_END_TIME_TITLE")
    }

}


