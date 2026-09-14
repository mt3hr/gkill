'use strict'

import { i18n } from '@/i18n'
import { KFTLStatementLine } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'

export class KFTLKCNumValueStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(context.get_next_statement_line_text()))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        try {
            this.parse_num_value()
            return i18n.global.t("KFTL_KC_NUM_VALUE_TITLE")
        } catch (_e: unknown) {
            return i18n.global.t("KFTL_KC_INVALID_NUM_VALUE_TITLE")
        }
    }
    private parse_num_value(): number {
        try {
            const num_value = Number.parseFloat(this.get_context().get_this_statement_line_text().trim()).valueOf()
            if (!num_value) {
                throw new Error(i18n.global.t("KFTL_KC_INVALID_NUM_VALUE_MESSAGE_TITLE"))
            }
            return num_value
        } catch (_e: unknown) {
            throw new Error(i18n.global.t("KFTL_KC_INVALID_NUM_VALUE_MESSAGE_TITLE"))
        }
    }
}


