'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLNlogTitleStatementLine } from './kftl-nlog-title-statement-line'
import { assert_is_not_meta_info_line, type KFTLNlogBlock } from './kftl-nlog-block'

export class KFTLNlogShopNameStatementLine extends KFTLStatementLine {

    private block: KFTLNlogBlock

    constructor(line_text: string, context: KFTLStatementLineContext, block: KFTLNlogBlock) {
        super(line_text, context)
        this.block = block
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor((line_text: string, context: KFTLStatementLineContext) => new KFTLNlogTitleStatementLine(line_text, context, block))
    }

    async apply_this_line_to_request_map(_request_map: KFTLRequestMap): Promise<void> {
        assert_is_not_meta_info_line(this.get_context().get_this_statement_line_text())
        this.block.shop_name = this.get_context().get_this_statement_line_text()
        return new Promise<void>((resolve) => resolve())

    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_NLOG_SHOP_NAME_TITLE")
    }

}
