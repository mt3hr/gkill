'use strict'

import moment from 'moment'
import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLMiReKyouRequest } from './kftl-mi-re-kyou-request'
import { KFTLMiReKyouTagStatementLine } from './kftl-mi-re-kyou-tag-statement-line'
import { KFTL_ASCII_TIMEIS_TIME_PREFIX, strip_prefix } from '../kftl-prefixes'

/**
 * リポストタスクの期日行。項目行はここで終わり。
 * このあとはタグ行を好きなだけ書けて、「～～」で閉じる
 */
export class KFTLMiReKyouLimitTimeStatementLine extends KFTLStatementLine {

    private request: KFTLMiReKyouRequest

    constructor(line_text: string, context: KFTLStatementLineContext, request: KFTLMiReKyouRequest, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        this.request = request
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLMiReKyouTagStatementLine.generate_after_last_field_constructor(context.get_next_statement_line_text(), request, prev_line_is_meta_info))
    }

    async apply_this_line_to_request_map(_request_map: KFTLRequestMap): Promise<void> {
        const time = moment(strip_prefix(this.get_context().get_this_statement_line_text(), "KFTL_TIMEIS_TIME_PREFIX", KFTL_ASCII_TIMEIS_TIME_PREFIX)).toDate()
        if (!Number.isNaN(time.getTime())) {
            this.request.set_limit_time(time)
        }
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        const line_text = this.get_context().get_this_statement_line_text()
        if (line_text == "" || line_text == "\n") {
            return i18n.global.t("KFTL_MI_NO_LIMIT_TIME_TITLE")
        }
        const time = moment(strip_prefix(line_text, "KFTL_TIMEIS_TIME_PREFIX", KFTL_ASCII_TIMEIS_TIME_PREFIX)).toDate()
        if (Number.isNaN(time.getTime())) {
            return i18n.global.t("KFTL_MI_INVALID_LIMIT_TIME_TITLE")
        }
        return i18n.global.t("KFTL_MI_LIMIT_TIME_TITLE")
    }
}
