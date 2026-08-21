'use strict'

import { parse_kftl_date_time } from '../kftl-date-time'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLPrototypeRequest } from '../kftl_prototype/kftl-prototype-request'
import type { KFTLTimeIsRequest } from './kftl-time-is-request'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import { i18n } from '@/i18n'
import { KFTL_ASCII_TIMEIS_TIME_PREFIX, strip_prefix } from '../kftl-prefixes'

export class KFTLTimeIsEndTimeStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(this.get_context().get_next_statement_line_text()))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        let request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLTimeIsRequest
        if (!request) {
            request_map.set(this.get_context().get_this_statement_line_target_id(), new KFTLPrototypeRequest(this.get_context().get_this_statement_line_target_id(), this.get_context()))
            request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLTimeIsRequest
        }
        const time = parse_kftl_date_time(strip_prefix(this.get_context().get_this_statement_line_text(), "KFTL_TIMEIS_TIME_PREFIX", KFTL_ASCII_TIMEIS_TIME_PREFIX))
        if (time === null) {
            throw new Error(i18n.global.t("KFTL_TIMEIS_INVALID_PARSE_TIME_ERROR_MESSAGE_TITLE"))
        }
        request.set_end_time(time)
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TIMEIS_END_TIME_LABEL_TITLE")
    }

}


