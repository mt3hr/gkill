'use strict'

import { i18n } from '@/i18n'
import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { KFTLMiReKyouNextLineConstructor } from './kftl-mi-re-kyou-tag-statement-line'

/**
 * 項目行を書き終えたあとの、ブロックの中の行。
 *
 * ここをタグ行にしてはいけない。行ラベルの先読み(KFTLStatement.generate_line_label_data)は
 * 「次の行のコンストラクタ」が無くなるまで空行を組み立て続けるので、
 * タグ行が自分自身を次に指すと「タグ」が先読み上限(50行)ぶん並ぶ。
 * Miの期日行のあとと同じく「**********」の行として並べる。
 *
 * 後置タグを書けなくするわけではない。実際にタグ行を書けば先読みではなく本物の行として
 * KFTLMiReKyouTagStatementLine が組み立てられるので、今までどおりMiReKyou自身にタグが付く。
 *
 * 次の行の決め方はブロックの中のままにする(呼び出し元から渡された generate_next を使う)。
 * 素の KFTLNoneStatementLine にすると「～～」が閉じる行ではなく
 * 新しいブロックの開始行として解釈され、ブロックを閉じられなくなる。
 */
export class KFTLMiReKyouNoneStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext, generate_next_constructor: { (next_line_text: string): KFTLMiReKyouNextLineConstructor }) {
        super(line_text, context)
        context.set_is_next_prototype(context.is_this_prototype())
        context.set_next_statement_line_target_id(context.get_this_statement_line_target_id())
        context.set_next_statement_line_constructor(generate_next_constructor(context.get_next_statement_line_text()))
    }

    async apply_this_line_to_request_map(_request_map: KFTLRequestMap): Promise<void> {
        const line_text = this.get_context().get_this_statement_line_text()
        if (line_text == "" || line_text == "\n") {
            return new Promise<void>((resolve) => resolve())
        }
        // 項目行を全部書き終えたあとに来られるのはタグ行か閉じる行だけ。
        // テキストは飲み込まずにおかしな行として出す。飲み込むと、閉じ忘れたときに
        // メモの本文が丸ごとリポストタスクに吸われてしまう
        throw new Error(i18n.global.t('KFTL_NONE_VALUE_IS_NOT_BLANK_MESSAGE_TITLE'))
    }

    get_label_name(_context: KFTLStatementLineContext): string {
        return i18n.global.t("KFTL_NONE_LABEL_TITLE")
    }
}
