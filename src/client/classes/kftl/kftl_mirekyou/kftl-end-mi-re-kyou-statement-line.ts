'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTL_ASCII_MI_REKYOU_SPLITTER_TITLE, matches_exact, normalize_wave_dash } from '../kftl-prefixes'

/**
 * リポストタスクのブロックを閉じる行(「～～」)。
 *
 * ここから先は再びタスク化する対象のKyou側に戻るので、閉じたあとに書いたタグは対象に付く。
 * 兄弟のクラスをひとつもimportしない(ブロック内の行が先読みでこのクラスを使うため、
 * ここから逆にimportすると循環する)。
 */
export class KFTLEndMiReKyouStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_next_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        if (prev_line_is_meta_info) {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_kmemo_constructor(context.get_next_statement_line_text()))
        } else {
            context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(context.get_next_statement_line_text()))
        }
    }

    async apply_this_line_to_request_map(_request_map: KFTLRequestMap): Promise<void> {
        // 閉じるだけ。リクエストへの書き込みは無い
        return new Promise<void>((resolve) => resolve())
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_MI_REKYOU_END_LABEL_TITLE")
    }

    static is_this_type(line_text: string): boolean {
        return matches_exact(normalize_wave_dash(line_text), "KFTL_MI_REKYOU_SPLITTER_TITLE", KFTL_ASCII_MI_REKYOU_SPLITTER_TITLE)
    }
}
