'use strict'

import moment from 'moment'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLRequest } from '../kftl-request'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import { KFTLPrototypeRequest } from '../kftl_prototype/kftl-prototype-request'
import { i18n } from '@/i18n'

export class KFTLRelatedTimeStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        if (prev_line_is_meta_info) {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_kmemo_constructor(context.get_next_statement_line_text()))
        } else {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(context.get_next_statement_line_text()))
        }
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        let request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLRequest
        if (!request) {
            request_map.set(this.get_context().get_this_statement_line_target_id(), new KFTLPrototypeRequest(this.get_context().get_this_statement_line_target_id(), this.get_context()))
            request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLRequest
        }
        const time = moment(this.get_context().get_this_statement_line_text().replace(i18n.global.t("KFTL_RELATED_TIME_PREFIX"), "")).toDate()
        if (Number.isNaN(time.getTime())) {
            throw new Error(i18n.global.t("KFTL_INVALID_PARSE_RELATED_TIME_ERROR_MESSAGE_TITLE"))
        }
        request.set_related_time(time)
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        const time = moment(this.get_context().get_this_statement_line_text().replace(i18n.global.t("KFTL_RELATED_TIME_PREFIX"), "")).toDate()
        if (Number.isNaN(time.getTime())) {
            return i18n.global.t("KFTL_INVALID_RELATED_TIME_TITLE")
        }
        return i18n.global.t("KFTL_RELATED_TIME_TITLE")
    }

    static is_this_type(line_text: string): boolean {
        return line_text.startsWith(i18n.global.t("KFTL_RELATED_TIME_PREFIX"))
    }

}