'use strict'

import type { KFTLStatementLineContext } from '@/classes/kftl/kftl-statement-line-context'
import { KFTLStatementLine } from '../../../kftl-statement-line'
import { GkillAPI } from '@/classes/api/gkill-api'
import { KFTLTimeIsEndByTagTagNameStatementLine } from './kftl-time-is-end-by-tag-tag-name-statement-line'
import { i18n } from '@/i18n'
import { KFTL_ASCII_TIMEIS_END_TAG_END_SPLITTER_TITLE, matches_exact } from '../../../kftl-prefixes'

export class KFTLStartTimeIsEndByTagStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        const target_id = context.is_this_prototype() && this.get_prev_line() ? this.get_prev_line()!.get_context().get_this_statement_line_target_id() : GkillAPI.get_gkill_api().generate_uuid()
        context.set_this_statement_line_target_id(target_id)
        context.set_next_statement_line_target_id(target_id)
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLTimeIsEndByTagTagNameStatementLine(line_text, context))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TIMEIS_TIMEIS_LABEL_TITLE")
    }

    static is_this_type(line_text: string): boolean {
        return matches_exact(line_text, "KFTL_TIMEIS_END_TAG_END_SPLITTER_TITLE", KFTL_ASCII_TIMEIS_END_TAG_END_SPLITTER_TITLE)
    }
}