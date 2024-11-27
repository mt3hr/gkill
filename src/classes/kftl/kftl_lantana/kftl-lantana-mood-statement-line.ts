'use strict'

import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLLantanaRequest } from './kftl-lantana-request'

export class KFTLLantanaMoodStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(context.get_next_statement_line_text()))
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        const mood = this.parse_mood()
        const lantana_request = request_map.get(this.get_context().get_this_statement_line_target_id()) as unknown as KFTLLantanaRequest
        lantana_request.set_mood(mood)
    }

    get_label_name(context: KFTLStatementLineContext): string {
        try {
            this.parse_mood()
            return "気分値"
        } catch (e: any) {
            return "変な気分値"
        }
    }
    private parse_mood(): number {
        try {
            const mood = Number.parseInt(this.get_context().get_this_statement_line_text().trim()).valueOf()
            if (!mood) {
                throw new Error("気分値が変です")
            }
            if (!(0 <= mood && mood <= 10)) {
                throw new Error("気分値が範囲外です")
            }
            return mood
        } catch (e: any) {
            throw new Error("気分値が変です")
        }
    }
}


