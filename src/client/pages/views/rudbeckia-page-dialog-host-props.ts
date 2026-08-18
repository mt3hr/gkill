'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { KyouChangeBus } from "@/classes/kyou-change-bus"

export interface RudbeckiaPageDialogHostProps extends GkillPropsBase {
    /** ポートの内容領域。各ウィンドウの既定サイズを決めるためだけに使う */
    app_content_height: number
    app_content_width: number
    application_config_load_failed: boolean
    /** 画面間の変更通知バス。そのまま各ウィンドウへ配る */
    kyou_change_bus: KyouChangeBus | null
}
