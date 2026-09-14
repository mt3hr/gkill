'use strict'

import { i18n } from '@/i18n'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLMiReKyouEstimateStartTimeStatementLine } from './kftl-mi-re-kyou-estimate-start-time-statement-line'
import { KFTLMiReKyouTagStatementLine } from './kftl-mi-re-kyou-tag-statement-line'

// リポストタスクの板名行。ラベルはMiと同じものを使う(意味が同一なため)
export class KFTLMiReKyouBoardNameStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLMiReKyouTagStatementLine.generate_next_constructor(context.get_next_statement_line_text(), prev_line_is_meta_info, (line_text: string, context: KFTLStatementLineContext) => new KFTLMiReKyouEstimateStartTimeStatementLine(line_text, context, prev_line_is_meta_info)))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_MI_BOARD_NAME_TITLE")
    }
}
