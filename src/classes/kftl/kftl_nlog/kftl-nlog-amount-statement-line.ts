'use strict'

import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLNlogRequest } from './kftl-nlog-request'

export class KFTLNlogAmountStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_nlog_constructor(this.get_context().get_next_statement_line_text()))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const amount = this.parse_amount()
        const nlog_request = request_map.get(this.get_context().get_this_statement_line_target_id()) as unknown as KFTLNlogRequest
        nlog_request.add_amount(amount)
    }

    get_label_name(context: KFTLStatementLineContext): string {
        try {
            const line_text = this.get_context().get_this_statement_line_text()
            if (line_text == "" || line_text == "\n") {
                return "金額"
            }
            const amount = this.parse_amount()
            if (0 < amount) {
                return "収入"
            } else {
                return "支出"
            }
        } catch (e: any) {
            return "変な金額"
        }
    }

    private parse_amount(): number {
        try {
            const amount = Number.parseInt(this.get_context().get_this_statement_line_text().trim())
            if (Number.isNaN(amount)) {
                throw new Error("金額が変です")
            }
            return amount
        } catch (e: any) {
            throw new Error("金額が変です")
        }
    }

}


