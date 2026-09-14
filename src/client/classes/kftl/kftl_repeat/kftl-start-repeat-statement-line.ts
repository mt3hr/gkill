'use strict'

import { i18n } from '@/i18n'
import { is_repeat_splitter } from '../kftl-prefixes'
import { KFTLStatementLine, type KFTLBlockReentryProvider } from '../kftl-statement-line'
import { KFTLStatementLineConstructorFactory } from '../kftl-statement-line-constructor-factory'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { generate_repeat_block_next_constructor, REPEAT_FIELD_CONDITION } from './kftl-repeat-block'
import { new_repeat_spec, type RepeatSpec } from './kftl-repeat-spec'

/**
 * 繰り返しブロック「？？」を開く行。
 *
 * タグ行・テキスト開始行と同じく**項目の位置を消費しない**。ブロック（タスク / リポストタスク /
 * 支出）の中に書いても、閉じたあとは block_reentry でブロックへ戻る。
 *
 * Mirrors: kftlStartRepeatStatementLine (src/server/gkill/api/kftl/kftl_repeat_lines.go)
 */
export class KFTLStartRepeatStatementLine extends KFTLStatementLine {

    private spec: RepeatSpec


    /** 「？？ 金 3」のように同じ行へ引数を書いたか。 */
    private written_with_argument: boolean

    constructor(line_text: string, context: KFTLStatementLineContext, prev_line_is_meta_info: boolean, block_reentry: KFTLBlockReentryProvider | null = null) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        this.spec = new_repeat_spec(context.get_kftl_statement_lines().length)
        this.written_with_argument = !is_repeat_splitter(line_text)

        if (this.written_with_argument) {
            // ブロックの連鎖を乗っ取らない。乗っ取ると後続の行まで繰り返し指定として読まれる
            if (block_reentry) {
                context.set_next_statement_line_constructor(block_reentry(context.get_next_statement_line_text()))
            } else if (prev_line_is_meta_info) {
                context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_kmemo_constructor(context.get_next_statement_line_text()))
            } else {
                context.set_next_statement_line_constructor(KFTLStatementLineConstructorFactory.get_instance().generate_none_constructor(context.get_next_statement_line_text()))
            }
            return
        }

        context.set_next_statement_line_constructor(generate_repeat_block_next_constructor(
            context.get_next_statement_line_text(), this.spec, REPEAT_FIELD_CONDITION, prev_line_is_meta_info, block_reentry))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        if (this.written_with_argument) {
            return i18n.global.t("KFTL_REPEAT_LABEL_TITLE")
        }
        // 回数が書かれていれば出す。「閉じ忘れて意図せず52件」に対する防御線。
        //
        // **実際に作られる件数までは出せない。** 候補の計算にはアンカー（レコードの日時欄）が
        // 必要だが、行ラベルの生成では request_map を作らないので手元に無い。
        // 2行目が終了日のときも同じ理由で件数が決まらないので、素のラベルに戻す
        if (this.spec.count > 0) {
            return i18n.global.t("KFTL_REPEAT_SUMMARY_LABEL_TITLE", { count: this.spec.count })
        }
        return i18n.global.t("KFTL_REPEAT_LABEL_TITLE")
    }

    get_spec(): RepeatSpec {
        return this.spec
    }

    static is_this_type(line_text: string): boolean {
        return is_repeat_splitter(line_text)
    }
}
