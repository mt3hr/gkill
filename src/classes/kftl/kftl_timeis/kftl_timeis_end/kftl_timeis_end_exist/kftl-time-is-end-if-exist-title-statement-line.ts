'use strict'

import type { KFTLRequestMap } from '@/classes/kftl/kftl-request-map'
import type { KFTLStatementLineContext } from '@/classes/kftl/kftl-statement-line-context'
import { KFTLStatementLine } from '../../../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '@/classes/kftl/kftl-statement-line-constructor-factory'
import type { KFTLTimeIsEndByTitleRequest } from '../kftl-time-is-end-by-title-request'
import { i18n } from '@/i18n'

export class KFTLTimeIsEndIfExistTitleStatementLine extends KFTLStatementLine {
    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(this.get_context().get_next_statement_line_text()))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLTimeIsEndByTitleRequest
        req.set_title(this.get_context().get_this_statement_line_text())
        req.set_error_when_target_does_not_exist(false)
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TIMEIS_TIMEIS_END_LABEL_TITLE")
    }

}