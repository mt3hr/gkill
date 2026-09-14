'use strict'

import { GkillAPI } from '@/classes/api/gkill-api'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { i18n } from '@/i18n'

export class KFTLKmemoStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        const target_id = (this.get_prev_line() && this.get_prev_line()?.get_context() && this.get_prev_line()?.get_context().is_this_prototype() || this.prev_line_is_kmemo_statement())
            ? this.get_prev_line()!.get_context().get_this_statement_line_target_id()
            : GkillAPI.get_gkill_api().generate_uuid()
        context.set_this_statement_line_target_id(target_id)
        context.set_next_statement_line_target_id(target_id)
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        if (this.prev_line_is_kmemo_statement()) {
            return ""
        }
        return i18n.global.t("KFTL_KMEMO_LABEL_TITLE")
    }

    private prev_line_is_kmemo_statement(): boolean {
        const lines = this.get_context().get_kftl_statement_lines()
        if (1 <= lines.length) {
            const prev_line = lines[lines.length - 1]
            if (prev_line == null) {
                return false
            }
            if (KFTLKmemoStatementLine.is_kmemo_statement_line(prev_line)) {
                return true
            }
        }
        return false
    }

    static is_this_type(_line_text: string): boolean {
        return true
    }

    private static is_kmemo_statement_line(statement_line: KFTLStatementLine): boolean {
        // クラス名ではなくコンストラクタの同一性で比較する。
        // minify するとクラス名は短縮され、別クラスが同じ名前になりうるため。
        return statement_line.constructor === KFTLKmemoStatementLine
    }

}


