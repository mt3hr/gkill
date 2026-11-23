'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

/**
 * HTML へ出力したファイルの中の rykv 画面。
 *
 * **読み取り専用であることを表すフラグは持たない。** 編集系の導線（画面切替メニュー・
 * 編集ダイアログ群・FAB・ヘルプ・設定）はこのビューがそもそも描かないので、
 * 「出力かどうか」を判定する場所が要らない。列の中身は `rykv-columns-view.vue` で、
 * 何ができるかはそこへリテラルで渡す。
 *
 * 出力時に「含めるもの」で選ばれなかった区画だけが下のフラグで決まる。
 * これは**モードではなく、焼き込んだ内容そのもの**（含めていない区画のデータは
 * ファイルの中に無い）。
 */
export interface ExportedRykvViewProps extends GkillPropsBase {
    app_title_bar_height: number
    app_content_height: number
    app_content_width: number
    // アプリバーに出す題名。出力時に利用者が付けた名前
    share_title: string
    // 集計（Dnote）を含めたか
    is_exported_with_dnote: boolean
    // マップを含めたか
    is_exported_with_map: boolean
    // 検索条件サイドバーを含めたか。含めた場合は焼いた記録の中での絞り込み専用
    is_exported_with_sidebar: boolean
}
