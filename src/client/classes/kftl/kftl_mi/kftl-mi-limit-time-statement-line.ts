'use strict'

import { parse_schedule_field_time } from '../kftl-schedule-field-time'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import { i18n } from '@/i18n'

export class KFTLMiLimitTimeStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(line_text)(line_text, context))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        const line_text = this.get_context().get_this_statement_line_text()
        if (line_text == "" || line_text == "\n") {
            return i18n.global.t("KFTL_MI_NO_LIMIT_TIME_TITLE")
        }
        try {
            if (parse_schedule_field_time(line_text) === null) {
                return i18n.global.t("KFTL_MI_INVALID_LIMIT_TIME_TITLE")
            }
        } catch (_e: unknown) {
            return i18n.global.t("KFTL_MI_INVALID_LIMIT_TIME_TITLE")
        }
        return i18n.global.t("KFTL_MI_LIMIT_TIME_TITLE")
    }

}


