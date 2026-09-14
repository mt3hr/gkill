'use strict'

import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLTimeIsEndTimeStatementLine } from './kftl-time-is-end-time-statement-line'
import { i18n } from '@/i18n'

export class KFTLTimeIsStartTimeStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLTimeIsEndTimeStatementLine(line_text, context))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TIMEIS_START_TIME_LABEL_TITLE")
    }
}


