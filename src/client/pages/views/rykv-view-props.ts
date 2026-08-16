'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

export interface RykvViewProps extends GkillPropsBase {
    app_title_bar_height: number
    app_content_height: number
    app_content_width: number
    is_shared_rykv_view: boolean
    share_title: string
    // ApplicationConfigの取得に失敗した。読み込み中オーバーレイを
    // スピナーからエラー表示＋再試行へ差し替えるために使う
    application_config_load_failed: boolean
}
