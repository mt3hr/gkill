'use strict'

import type { KyouChangeChannel } from "@/classes/kyou-change-bus"
import type { GkillPropsBase } from "./gkill-props-base"

export interface PlaingTimeIsViewProps extends GkillPropsBase {
    app_content_height: number
    app_content_width: number
    // ポート(rudbeckia)のフローティングダイアログの中、または打刻メモ帳ダイアログの中で
    // 描かれている。意味は rykv-view-props.ts の同名 prop と同じ
    is_hosted_in_dialog: boolean
    // 画面をまたいだ変更通知の口。null が「単独ページとして動く」＝ publish も購読もしない。
    // ポート(rudbeckia)で開いたときだけ非 null が入る
    kyou_change_channel: KyouChangeChannel | null
}
