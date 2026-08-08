'use strict'

import type { Notification } from "@/classes/datas/notification"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface AddNotificationForAddMiViewProps extends KyouViewPropsBase {
    default_notification: Notification | null
    // 保存は親(add-mi-view / add-mi-re-kyou-view)がまとめて行うので、
    // 送信中かどうかも親から受け取る
    is_readonly: boolean
}
