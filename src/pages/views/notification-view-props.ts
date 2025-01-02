'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"
import type { Notification } from "@/classes/datas/notification"

export interface NotificationViewProps extends GkillPropsBase {
    notification : Notification
    last_added_tag: string
    kyou: Kyou
    highlight_targets: Array<InfoIdentifier>
    enable_context_menu: boolean
    enable_dialog: boolean
}
