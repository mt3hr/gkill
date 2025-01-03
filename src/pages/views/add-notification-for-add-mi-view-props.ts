'use strict'

import type { Notification } from "@/classes/datas/notification"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface AddNotificationForAddMiViewProps extends KyouViewPropsBase {
    default_notification: Notification | null
}
