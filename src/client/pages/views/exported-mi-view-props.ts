'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

/**
 * HTML へ出力したファイルの中の mi（タスク画面）。
 *
 * **読み取り専用であることを表すフラグは持たない。** 編集系の導線（画面切替メニュー・
 * 編集ダイアログ群・FAB・ヘルプ・設定）はこのビューがそもそも描かないので、
 * 「出力かどうか」を判定する場所が要らない。列の中身は `mi-columns-view.vue` で、
 * 何ができるかはそこへリテラルで渡す。
 *
 * 出力時に「含めるもの」で選ばれなかった区画だけが下のフラグで決まる。
 * これは**モードではなく、焼き込んだ内容そのもの**（含めていない区画のデータは
 * ファイルの中に無い）。rykv と違って集計・マップのフラグが無いのは、
 * mi にその区画が無いため。
 */
export interface ExportedMiViewProps extends GkillPropsBase {
    app_title_bar_height: number
    app_content_height: number
    app_content_width: number
    // アプリバーに出す題名。出力時に利用者が付けた名前
    share_title: string
    // 検索条件サイドバーを含めたか。含めた場合は焼いた記録の中での絞り込み専用
    is_exported_with_sidebar: boolean
}
