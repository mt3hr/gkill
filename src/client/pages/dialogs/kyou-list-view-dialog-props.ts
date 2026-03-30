'use strict'

import type { InfoIdentifier } from "@/classes/datas/info-identifier"
import type { GkillPropsBase } from "../views/gkill-props-base"

export interface KyouListViewDialogProps extends GkillPropsBase {
    highlight_targets: Array<InfoIdentifier>
    list_height: number
    enable_context_menu: boolean
    enable_dialog: boolean
    force_show_latest_kyou_info: boolean
    show_rep_name: boolean
}
