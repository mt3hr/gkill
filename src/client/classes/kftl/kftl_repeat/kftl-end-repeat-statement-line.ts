'use strict'

import { i18n } from '@/i18n'
import { KFTLStatementLine, type KFTLBlockReentryProvider } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { RepeatSpec } from './kftl-repeat-spec'

/**
 * 繰り返しブロック「？？」を閉じる行。
 *
 * テキストブロックの終了行と同じ戻り方をする。ブロックの中なら block_reentry、
 * 外なら kmemo / none。
 *
 * Mirrors: kftlEndRepeatStatementLine (src/server/gkill/api/kftl/kftl_repeat_lines.go)
 */
export class KFTLEndRepeatStatementLine extends KFTLStatementLine {

    private spec: RepeatSpec

    constructor(line_text: string, context: KFTLStatementLineContext, spec: RepeatSpec, prev_line_is_meta_info: boolean, block_reentry: KFTLBlockReentryProvider | null) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        this.spec = spec
        if (block_reentry) {
            context.set_next_statement_line_constructor(block_reentry(context.get_next_statement_line_text()))
        } else if (prev_line_is_meta_info) {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_kmemo_constructor(context.get_next_statement_line_text()))
        } else {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(context.get_next_statement_line_text()))
        }
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_REPEAT_END_LABEL_TITLE")
    }
}
