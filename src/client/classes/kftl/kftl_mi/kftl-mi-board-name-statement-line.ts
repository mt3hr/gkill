'use strict'

import { i18n } from '@/i18n'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { generate_mi_block_next_constructor } from './kftl-mi-block'
import { KFTLMiEstimateStartTimeStatementLine } from './kftl-mi-estimate-start-time-statement-line'

export class KFTLMiBoardNameStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        context.set_next_statement_line_constructor(generate_mi_block_next_constructor(context.get_next_statement_line_text(), (line_text: string, context: KFTLStatementLineContext) => new KFTLMiEstimateStartTimeStatementLine(line_text, context)))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_MI_BOARD_NAME_TITLE")
    }

}