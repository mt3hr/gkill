'use strict'

import { GkillAPI } from '@/classes/api/gkill-api'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'

export class KFTLSplitAndNextSecondStatementLine extends KFTLStatementLine {
    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_this_statement_line_target_id(GkillAPI.get_gkill_api().generate_uuid())
        context.set_is_this_prototype(true)
        context.set_is_next_prototype(true)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => KFTLStatementLineConstructorFactory.get_instance().generate_kmemo_constructor(line_text)(line_text, context))
        context.set_next_statement_line_target_id(null)
    }

    async apply_this_line_to_request_map(_requet_map: KFTLRequestMap): Promise<void> {
        return
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return "======"
    }

    static is_this_type(line_text: string): boolean {
        return line_text == "、、"
    }
}


