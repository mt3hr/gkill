'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ExportRykvHTMLViewProps extends GkillPropsBase {
    /**
     * 表示中のすべての列の検索条件。画面に出ている順。
     *
     * **サイドバーが生成したものをそのまま渡すこと**（タグや記録保管場所の
     * チェック状態が入っている）。条件を削ってから渡すと、出力先のサイドバーが
     * 初めに生成した条件と食い違った瞬間にその列が空になる。
     */
    columns: Array<FindKyouQuery>

    /** フォーカスされている列の位置。無ければ -1 */
    focused_column_index: number
}
