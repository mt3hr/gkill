'use strict'

import { i18n } from '@/i18n'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { KFTLMiReKyouBoardNameStatementLine } from './kftl-mi-re-kyou-board-name-statement-line'
import { KFTLMiReKyouTagStatementLine } from './kftl-mi-re-kyou-tag-statement-line'
import { KFTL_ASCII_MI_REKYOU_SPLITTER_TITLE, matches_exact, normalize_wave_dash } from '../kftl-prefixes'

/**
 * リポストタスクのブロックを開く行(「～～」)。
 *
 * 同じレコードで書いたKyouをタスク化するので、バケツリレーされてきたtarget_idを
 * そのまま対象として使う。MiReKyou自身は別のidを持ち、そちらがrequest_mapのキーになる
 * (Miと違ってここでtarget_idを作り直さないこと。作り直すと対象を見失う)。
 */
export class KFTLStartMiReKyouStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean) {
        super(line_text, context)
        const target_id = context.get_this_statement_line_target_id()
        // ブロックはKyou本体ではなく付随情報なので、プロトタイプかどうかを次の行へ伝える。
        // 伝えないと「ブロックを先に書いてあとからメモを書く」並びでメモが別のidを引き当てて、
        // このMiReKyouの対象が消える
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(target_id)
        context.set_next_statement_line_constructor(KFTLMiReKyouTagStatementLine.generate_next_constructor(context.get_next_statement_line_text(), prev_line_is_meta_info, (line_text: string, context: KFTLStatementLineContext) => new KFTLMiReKyouBoardNameStatementLine(line_text, context, prev_line_is_meta_info)))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_MI_REKYOU_START_LABEL_TITLE")
    }

    static is_this_type(line_text: string): boolean {
        return matches_exact(normalize_wave_dash(line_text), "KFTL_MI_REKYOU_SPLITTER_TITLE", KFTL_ASCII_MI_REKYOU_SPLITTER_TITLE)
    }
}
