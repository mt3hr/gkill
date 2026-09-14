'use strict'

import { parse_schedule_field_time } from '../kftl-schedule-field-time'
import { i18n } from '@/i18n'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLMiReKyouTagStatementLine } from './kftl-mi-re-kyou-tag-statement-line'

/**
 * リポストタスクの期日行。項目行はここで終わり。
 * このあとはタグ行を好きなだけ書けて、「～～」で閉じる
 */
export class KFTLMiReKyouLimitTimeStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLMiReKyouTagStatementLine.generate_after_last_field_constructor(context.get_next_statement_line_text(), prev_line_is_meta_info))
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
