'use strict'

import { i18n } from '@/i18n'
import { KFTLStatementLine, type KFTLBlockReentryProvider } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import {
    generate_repeat_block_next_constructor,
    REPEAT_FIELD_ADD_IF_EXISTS,
    REPEAT_FIELD_CONDITION,
    REPEAT_FIELD_COUNT,
    REPEAT_FIELD_ORIGIN,
} from './kftl-repeat-block'
import {
    parse_repeat_add_if_exists,
    parse_repeat_condition,
    parse_repeat_count_or_until,
    parse_repeat_origin,
    type RepeatSpec,
} from './kftl-repeat-spec'

/**
 * 繰り返しブロック「？？」の中の1行。位置で意味が決まる。
 *
 * Mirrors: kftlRepeatFieldStatementLine (src/server/gkill/api/kftl/kftl_repeat_lines.go)
 */
export class KFTLRepeatFieldStatementLine extends KFTLStatementLine {

    private spec: RepeatSpec

    private index: number

    constructor(line_text: string, context: KFTLStatementLineContext, spec: RepeatSpec, index: number, prev_line_is_meta_info: boolean, block_reentry: KFTLBlockReentryProvider | null) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        this.spec = spec
        this.index = index
        context.set_next_statement_line_constructor(generate_repeat_block_next_constructor(
            context.get_next_statement_line_text(), spec, index + 1, prev_line_is_meta_info, block_reentry))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        const line_text = this.get_statement_line_text()
        switch (this.index) {
            case REPEAT_FIELD_CONDITION:
                return label_of("KFTL_REPEAT_CONDITION_TITLE", "KFTL_REPEAT_INVALID_CONDITION_TITLE", () => parse_repeat_condition(line_text))
            case REPEAT_FIELD_COUNT:
                return label_of("KFTL_REPEAT_COUNT_TITLE", "KFTL_REPEAT_INVALID_COUNT_TITLE", () => parse_repeat_count_or_until(line_text))
            case REPEAT_FIELD_ADD_IF_EXISTS:
                return label_of("KFTL_REPEAT_ADD_IF_EXISTS_TITLE", "KFTL_REPEAT_INVALID_ADD_IF_EXISTS_TITLE", () => parse_repeat_add_if_exists(line_text))
            case REPEAT_FIELD_ORIGIN:
                return label_of("KFTL_REPEAT_ORIGIN_TITLE", "KFTL_REPEAT_INVALID_ORIGIN_TITLE", () => parse_repeat_origin(line_text))
        }
        // 4行を書き終えたあとの位置。**「繰り返し↓」を返してはいけない。**
        // 行ラベルの先読み(KFTLStatement.generate_line_label_data)は「次の行のコンストラクタ」がある限り
        // 空行を上限(50行)ぶん組み立てるので、ここが「繰り返し↓」だとラベルの列がそれで埋まる。
        // Miの期日行のあと・「～～」の項目行のあとと同じく「**********」を並べる
        return i18n.global.t("KFTL_NONE_LABEL_TITLE")
    }
}

/** apply と同じパーサでラベルを決める。片方だけ直すと「保存はできるのに変な表示」になる。 */
function label_of(valid_key: string, invalid_key: string, parse: () => void): string {
    try {
        parse()
        return i18n.global.t(valid_key)
    } catch (_e: unknown) {
        return i18n.global.t(invalid_key)
    }
}
