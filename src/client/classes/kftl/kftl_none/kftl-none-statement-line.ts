'use strict'

import { i18n } from '@/i18n'
import { KFTLStatementLine } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'

export class KFTLNoneStatementLine extends KFTLStatementLine {
    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_is_next_prototype(true)
        context.set_is_this_prototype(true)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(line_text)(line_text, context))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_NONE_LABEL_TITLE")
    }
}


