'use strict'

import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLLantanaRequest } from './kftl-lantana-request'

export class KFTLStartLantanaStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const req = new KFTLLantanaRequest(this.get_context().get_this_statement_line_target_id(), this.get_context())
        request_map.set(this.get_context().get_this_statement_line_target_id(), req)
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(context: KFTLStatementLineContext): string {
        return "lantana"
    }

    static is_this_type(line_text: string): boolean {
        return line_text == "ーら"
    }

}


