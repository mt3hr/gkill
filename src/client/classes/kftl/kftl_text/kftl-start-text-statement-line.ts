'use strict'

import { KFTLStatementLine, type KFTLBlockReentryProvider } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLTextStatementLine } from './kftl-text-statement-line'
import { i18n } from '@/i18n'
import { KFTL_ASCII_TEXT_SPLITTER_TITLE, matches_exact } from '../kftl-prefixes'

export class KFTLStartTextStatementLine extends KFTLStatementLine {

    // block_reentry は本文行を経由して終了行まで運ばれる。出口を決めるのは終了行なので、
    // ここで使わなくても最後まで持ち回る必要がある
    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean, block_reentry: KFTLBlockReentryProvider | null = null) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLTextStatementLine(line_text, context, prev_line_is_meta_info, block_reentry))
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())

    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TEXT_START_LABEL_TITLE")
    }

    static is_this_type(line_text: string): boolean {
        return matches_exact(line_text, "KFTL_TEXT_SPLITTER_TITLE", KFTL_ASCII_TEXT_SPLITTER_TITLE)
    }

}


