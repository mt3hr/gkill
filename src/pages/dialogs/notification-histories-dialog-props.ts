'use strict'

import type { Notification } from "@/classes/datas/notification"
import type { KyouViewPropsBase } from "../views/kyou-view-props-base"

export interface NotificationHistoriesDialogProps extends KyouViewPropsBase {
    notification: Notification
}
