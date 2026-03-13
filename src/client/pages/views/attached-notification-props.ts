'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"
import { Notification } from "@/classes/datas/notification"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface AttachedNotificationProps extends GkillPropsBase {
    notification: Notification
    kyou: Kyou
    highlight_targets: Array<InfoIdentifier>
    enable_context_menu: boolean
    enable_dialog: boolean
}
