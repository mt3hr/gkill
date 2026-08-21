'use strict'

import { parse_kftl_date_time } from '../kftl-date-time'
import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTL_ASCII_RELATED_TIME_PREFIX, strip_prefix } from '../kftl-prefixes'
import { generate_nlog_block_next_constructor, type KFTLNlogBlock } from './kftl-nlog-block'

/**
 * 支出ブロックの中の関連時刻行。
 *
 * 付け先は直前の支払いではなくブロックで、ブロックの中のどこに書いても全支払いに効く。
 * 支払いは1組ずつ別のリクエストになったが、関連時刻だけは「その買い物をした時刻」なので
 * ブロック単位のままにしている。
 *
 * リクエストの実行は行の解釈が全部終わったあとなので、ブロックのどの位置に書いても
 * (自分より前に作られた支払いにも)効く。
 */
export class KFTLNlogRelatedTimeStatementLine extends KFTLStatementLine {

    private block: KFTLNlogBlock

    constructor(line_text: string, context: KFTLStatementLineContext, block: KFTLNlogBlock) {
        super(line_text, context)
        this.block = block
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(generate_nlog_block_next_constructor(context.get_next_statement_line_text(), block))
    }

    async apply_this_line_to_request_map(_request_map: KFTLRequestMap): Promise<void> {
        this.block.related_time = this.parse_related_time()
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        try {
            this.parse_related_time()
        } catch (_e: unknown) {
            return i18n.global.t("KFTL_INVALID_RELATED_TIME_TITLE")
        }
        return i18n.global.t("KFTL_RELATED_TIME_TITLE")
    }

    private parse_related_time(): Date {
        const time = parse_kftl_date_time(strip_prefix(this.get_context().get_this_statement_line_text(), "KFTL_RELATED_TIME_PREFIX", KFTL_ASCII_RELATED_TIME_PREFIX))
        if (time === null) {
            throw new Error(i18n.global.t("KFTL_INVALID_PARSE_RELATED_TIME_ERROR_MESSAGE_TITLE"))
        }
        return time
    }

}
