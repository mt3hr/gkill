'use strict'

import type { KFTLStatementLineContext } from '@/classes/kftl/kftl-statement-line-context'
import { KFTLStatementLine } from '../../../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '@/classes/kftl/kftl-statement-line-constructor-factory'
import { i18n } from '@/i18n'

export class KFTLTimeIsEndByTagIfExistTagNameStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(this.get_context().get_next_statement_line_text()))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TIMEIS_TIMEIS_END_LABEL_TITLE")
    }
}


