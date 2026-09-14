'use strict'

import { i18n } from '@/i18n'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLNlogTitleStatementLine } from './kftl-nlog-title-statement-line'
import type { KFTLNlogBlock } from './kftl-nlog-block'

export class KFTLNlogShopNameStatementLine extends KFTLStatementLine {

    private block: KFTLNlogBlock

    constructor(line_text: string, context: KFTLStatementLineContext, block: KFTLNlogBlock) {
        super(line_text, context)
        this.block = block
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLNlogTitleStatementLine(line_text, context, block))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_NLOG_SHOP_NAME_TITLE")
    }

}
