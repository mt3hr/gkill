'use strict'

import type { KFTLRequest } from '../kftl-request'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLEndTextStatementLine } from './kftl-end-text-statement-line'

export class KFTLTextStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_constructor(KFTLTextStatementLine.generate_wait_end_of_text_constructor(this.get_context().get_next_statement_line_text(), prev_line_is_meta_info))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLRequest
        const text_id = request.get_current_text_id()
        request.add_text_line(text_id!!, this.get_context().get_this_statement_line_text())
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return ""
    }

    static generate_wait_end_of_text_constructor(line_text: string, prototype: boolean): { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } {
        if (KFTLEndTextStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => new KFTLEndTextStatementLine(line_text, context, prototype)
        }
        return (line_text: string, context: KFTLStatementLineContext) => new KFTLTextStatementLine(line_text, context, prototype)
    }
}


