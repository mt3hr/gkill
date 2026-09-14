'use strict'

import { KFTLStatementLine } from '../../kftl-statement-line'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { KFTLStatementLineConstructorFactory } from '../../kftl-statement-line-constructor-factory'
import { i18n } from '@/i18n'

export class KFTLTimeIsEndTitleStatementLine extends KFTLStatementLine {
    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(this.get_context().get_next_statement_line_text()))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TIMEIS_TIMEIS_END_LABEL_TITLE")
    }
}


