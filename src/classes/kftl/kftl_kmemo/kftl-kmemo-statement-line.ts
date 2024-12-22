'use strict'

import { GkillAPI } from '@/classes/api/gkill-api'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLKmemoRequest } from './kftl-kmemo-request'

export class KFTLKmemoStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        const target_id = (this.get_prev_line() && this.get_prev_line()?.get_context() && this.get_prev_line()?.get_context().is_this_prototype() || this.prev_line_is_kmemo_statement())
            ? this.get_prev_line()!!.get_context().get_this_statement_line_target_id() 
            : GkillAPI.get_instance().generate_uuid()
        context.set_this_statement_line_target_id(target_id)
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        let target_id = this.get_context().get_this_statement_line_target_id()
        let kmemo_request: KFTLKmemoRequest
        try {
            kmemo_request = request_map.get(target_id) as KFTLKmemoRequest
            kmemo_request.add_kmemo_line(this.get_statement_line_text())
        } catch (e: any) {
            kmemo_request = new KFTLKmemoRequest(this.get_context().get_this_statement_line_target_id(), this.get_context())
            kmemo_request.add_kmemo_line(this.get_statement_line_text())
            request_map.set(target_id, kmemo_request)
        }
    }

    get_label_name(context: KFTLStatementLineContext): string {
        if (this.prev_line_is_kmemo_statement()) {
            return ""
        }
        return "kmemo"
    }

    private prev_line_is_kmemo_statement(): boolean {
        const lines = this.get_context().get_kftl_statement_lines()
        if (1 <= lines.length) {
            const prev_line = lines[lines.length - 1]
            if (prev_line == null) {
                return false
            }
            if (KFTLKmemoStatementLine.is_kmemo_statement_line(prev_line)) {
                return true
            }
        }
        return false
    }

    static is_this_type(line_text: string): boolean {
        return true
    }

    private static is_kmemo_statement_line(statement_line: any): boolean {
        return statement_line.constructor.name == KFTLKmemoStatementLine.name
    }

}


