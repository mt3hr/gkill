'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequest } from '../kftl-request'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'

export class KFTLEndTextStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_next_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        if (prev_line_is_meta_info) {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_kmemo_constructor(context.get_next_statement_line_text()))
        } else {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(context.get_next_statement_line_text()))
        }
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLRequest
        request.set_current_text_id(null)
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TEXT_END_LABEL_TITLE")
    }

    static is_this_type(line_text: string): boolean {
        return line_text == i18n.global.t("KFTL_TEXT_SPLITTER_TITLE")
    }
}


