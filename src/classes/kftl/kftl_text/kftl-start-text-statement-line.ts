'use strict'

import { GkillAPI } from '@/classes/api/gkill-api'
import type { KFTLRequest } from '../kftl-request'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLPrototypeRequest } from '../kftl_prototype/kftl-prototype-request'
import { KFTLTextStatementLine } from './kftl-text-statement-line'
import { i18n } from '@/i18n'

export class KFTLStartTextStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLTextStatementLine(line_text, context, prev_line_is_meta_info))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())

    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        let request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLRequest
        if (!request) {
            request_map.set(this.get_context().get_this_statement_line_target_id(), new KFTLPrototypeRequest(this.get_context().get_this_statement_line_target_id(), this.get_context()))
            request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLRequest
        }
        request.set_current_text_id(GkillAPI.get_gkill_api().generate_uuid())
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TEXT_START_LABEL_TITLE")
    }

    static is_this_type(line_text: string): boolean {
        return line_text == i18n.global.t("KFTL_TEXT_SPLITTER_TITLE")
    }

}


