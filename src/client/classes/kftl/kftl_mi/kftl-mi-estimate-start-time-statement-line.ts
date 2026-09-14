'use strict'

import { parse_schedule_field_time } from '../kftl-schedule-field-time'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { generate_mi_block_next_constructor } from './kftl-mi-block'
import { KFTLMiEstimateEndTimeStatementLine } from './kftl-mi-estimate-end-time-statement-line'
import { i18n } from '@/i18n'

export class KFTLMiEstimateStartTimeStatementLine extends KFTLStatementLine {
    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_constructor(generate_mi_block_next_constructor(context.get_next_statement_line_text(), (line_text: string, context: KFTLStatementLineContext) => new KFTLMiEstimateEndTimeStatementLine(line_text, context)))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())

    }

    get_label_name(_context: KFTLStatementLineContext): string {
        const line_text = this.get_context().get_this_statement_line_text()
        if (line_text == "" || line_text == "\n") {
            return i18n.global.t("KFTL_MI_NO_ESTIMATE_START_TIME_TITLE")
        }
        try {
            if (parse_schedule_field_time(line_text) === null) {
                return i18n.global.t("KFTL_MI_INVALID_ESTIMATE_START_TIME_TITLE")
            }
        } catch (_e: unknown) {
            return i18n.global.t("KFTL_MI_INVALID_ESTIMATE_START_TIME_TITLE")
        }
        return i18n.global.t("KFTL_MI_ESTIMATE_START_TIME_TITLE")
    }
}


