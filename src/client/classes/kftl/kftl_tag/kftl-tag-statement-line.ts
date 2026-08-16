'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequest } from '../kftl-request'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine, type KFTLBlockReentryProvider } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLPrototypeRequest } from '../kftl_prototype/kftl-prototype-request'
import { KFTL_ASCII_TAG_PREFIX, matches_prefix, split_tags, strip_prefix } from '../kftl-prefixes'

export class KFTLTagStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean, block_reentry: KFTLBlockReentryProvider | null = null) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        if (block_reentry) {
            // ブロックの中のタグ行。次の行もブロックの中として解釈する
            context.set_next_statement_line_constructor(block_reentry(context.get_next_statement_line_text()))
        } else if (prev_line_is_meta_info) {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_kmemo_constructor(context.get_next_statement_line_text()))
        } else {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(context.get_next_statement_line_text()))
        }
    }

    async apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void> {
        let request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLRequest
        if (!request) {
            request_map.set(this.get_context().get_this_statement_line_target_id(), new KFTLPrototypeRequest(this.get_context().get_this_statement_line_target_id(), this.get_context()))
            request = request_map.get(this.get_context().get_this_statement_line_target_id()) as KFTLRequest
        }
        const tags: Array<string> = split_tags(strip_prefix(this.get_statement_line_text(), "KFTL_TAG_PREFIX", KFTL_ASCII_TAG_PREFIX))
        tags.forEach(tag => {
            request.add_tag(tag)
        })
        return new Promise<void>((resolve) => resolve())

    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_TAG_LABEL_TITLE")
    }

    static is_this_type(line_text: string): boolean {
        return matches_prefix(line_text, "KFTL_TAG_PREFIX", KFTL_ASCII_TAG_PREFIX)
    }

}


