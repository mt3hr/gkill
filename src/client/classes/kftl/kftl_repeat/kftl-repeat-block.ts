'use strict'

import { is_repeat_splitter } from '../kftl-prefixes'
import type { KFTLBlockReentryProvider, KFTLStatementLineConstructor } from '../kftl-statement-line'
import { KFTLEndRepeatStatementLine } from './kftl-end-repeat-statement-line'
import { KFTLRepeatFieldStatementLine } from './kftl-repeat-field-statement-line'
import type { RepeatSpec } from './kftl-repeat-spec'

/**
 * 繰り返しブロック「？？」の中の行の位置。
 * 索引でしか区別しないので、順序を変えるときは Go 側と一緒に変えること。
 *
 * Mirrors: repeatFieldCondition ほか (src/server/gkill/api/kftl/kftl_repeat_lines.go)
 */
export const REPEAT_FIELD_CONDITION = 0
export const REPEAT_FIELD_COUNT = 1
export const REPEAT_FIELD_ADD_IF_EXISTS = 2
export const REPEAT_FIELD_ORIGIN = 3

/**
 * 「？？」ブロックの中の「次の行」を決める先読み。
 *
 * **閉じる行を最優先で見る**ので、3行目・4行目を省いて早く閉じられる。
 *
 * Mirrors: generateRepeatBlockNextConstructor (kftl_repeat_lines.go)
 */
export function generate_repeat_block_next_constructor(
    next_line_text: string,
    spec: RepeatSpec,
    index: number,
    prev_line_is_meta_info: boolean,
    block_reentry: KFTLBlockReentryProvider | null,
): KFTLStatementLineConstructor {
    if (is_repeat_splitter(next_line_text)) {
        return (line_text: string, context) => new KFTLEndRepeatStatementLine(line_text, context, spec, prev_line_is_meta_info, block_reentry)
    }
    return (line_text: string, context) => new KFTLRepeatFieldStatementLine(line_text, context, spec, index, prev_line_is_meta_info, block_reentry)
}
