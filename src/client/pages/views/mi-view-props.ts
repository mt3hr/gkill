'use strict'

import type { KyouChangeChannel } from "@/classes/kyou-change-bus"
import type { GkillPropsBase } from "./gkill-props-base"

export interface MiViewProps extends GkillPropsBase {
    app_title_bar_height: number
    app_content_height: number
    app_content_width: number
    // ApplicationConfigの取得に失敗した。読み込み中オーバーレイを
    // スピナーからエラー表示＋再試行へ差し替えるために使う
    application_config_load_failed: boolean
    // ポート(rudbeckia)のフローティングダイアログの中で描かれている。
    // 意味は rykv-view-props.ts の同名 prop と同じ（対称実装）
    is_hosted_in_dialog: boolean
    // 列の検索条件とスクロール位置の保存キーの枝番。
    // 意味は rykv-view-props.ts の同名 prop と同じ（対称実装）
    column_state_instance_key: string
    // 画面をまたいだ変更通知の口。null が「単独ページとして動く」＝ publish も購読もしない。
    // ポート(rudbeckia)で開いたときだけ非 null が入る
    kyou_change_channel: KyouChangeChannel | null
}
