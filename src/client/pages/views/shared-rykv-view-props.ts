'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

/**
 * 共有された rykv 画面。
 *
 * **モードのフラグを持たない。** 読み取り専用であることは、子へ渡す値を
 * リテラル（`:enable_dialog="false"` など）で書くことで表す（shared-mi-view と同じ作法）。
 * 通常の rykv（rykv-view）とは別のビューで、列の機構も検索も持たない ——
 * 共有は「渡された検索結果を1列で見せるだけ」なので、列・検索・列状態の保存は要らない。
 */
export interface SharedRykvViewProps extends GkillPropsBase {
    app_title_bar_height: number
    app_content_height: number
    app_content_width: number
    share_title: string
}
