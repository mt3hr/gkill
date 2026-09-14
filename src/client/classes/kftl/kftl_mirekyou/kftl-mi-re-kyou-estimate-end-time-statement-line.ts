'use strict'

import { parse_schedule_field_time } from '../kftl-schedule-field-time'
import { i18n } from '@/i18n'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLMiReKyouLimitTimeStatementLine } from './kftl-mi-re-kyou-limit-time-statement-line'
import { KFTLMiReKyouTagStatementLine } from './kftl-mi-re-kyou-tag-statement-line'

// リポストタスクの見積終了日時行。「？」/「?」を付けると行エラーになる
export class KFTLMiReKyouEstimateEndTimeStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLMiReKyouTagStatementLine.generate_next_constructor(context.get_next_statement_line_text(), prev_line_is_meta_info, (line_text: string, context: KFTLStatementLineContext) => new KFTLMiReKyouLimitTimeStatementLine(line_text, context, prev_line_is_meta_info)))
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
