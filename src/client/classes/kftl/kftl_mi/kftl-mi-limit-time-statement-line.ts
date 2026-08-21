'use strict'

import { parse_kftl_date_time } from '../kftl-date-time'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLMiRequest } from './kftl-mi-request'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import { i18n } from '@/i18n'
import { KFTL_ASCII_TIMEIS_TIME_PREFIX, strip_prefix } from '../kftl-prefixes'

export class KFTLMiLimitTimeStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(line_text)(line_text, context))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const request = request_map.get(this.get_context().get_this_statement_line_target_id()) as unknown as KFTLMiRequest
        const time = parse_kftl_date_time(strip_prefix(this.get_context().get_this_statement_line_text(), "KFTL_TIMEIS_TIME_PREFIX", KFTL_ASCII_TIMEIS_TIME_PREFIX))
        if (time !== null) {
            request.set_limit_time(time)
        }
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        const line_text = this.get_context().get_this_statement_line_text()
        if (line_text == "" || line_text == "\n") {
            return i18n.global.t("KFTL_MI_NO_LIMIT_TIME_TITLE")
        }
        const time = parse_kftl_date_time(strip_prefix(line_text, "KFTL_TIMEIS_TIME_PREFIX", KFTL_ASCII_TIMEIS_TIME_PREFIX))
        if (time === null) {
            return i18n.global.t("KFTL_MI_INVALID_LIMIT_TIME_TITLE")
        }
        return i18n.global.t("KFTL_MI_LIMIT_TIME_TITLE")
    }

}


