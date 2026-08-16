'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLMiReKyouRequest } from './kftl-mi-re-kyou-request'
import { KFTLMiReKyouEstimateStartTimeStatementLine } from './kftl-mi-re-kyou-estimate-start-time-statement-line'
import { KFTLMiReKyouTagStatementLine } from './kftl-mi-re-kyou-tag-statement-line'

// リポストタスクの板名行。ラベルはMiと同じものを使う(意味が同一なため)
export class KFTLMiReKyouBoardNameStatementLine extends KFTLStatementLine {

    private request: KFTLMiReKyouRequest

    constructor(line_text: string, context: KFTLStatementLineContext, request: KFTLMiReKyouRequest, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        this.request = request
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(KFTLMiReKyouTagStatementLine.generate_next_constructor(context.get_next_statement_line_text(), request, prev_line_is_meta_info, (line_text: string, context: KFTLStatementLineContext) => new KFTLMiReKyouEstimateStartTimeStatementLine(line_text, context, request, prev_line_is_meta_info)))
    }

    async apply_this_line_to_request_map(_request_map: KFTLRequestMap): Promise<void> {
        this.request.set_board_name(this.get_context().get_this_statement_line_text())
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_MI_BOARD_NAME_TITLE")
    }
}
