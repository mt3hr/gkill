'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"
import type { Text } from "@/classes/datas/text"
import type { Notification } from "@/classes/datas/notification"
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"

export interface KyouViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'registered_kyou', registered_kyou: Kyou): void
    (e: 'updated_kyou', updated_kyou: Kyou): void
    (e: 'deleted_kyou', deleted_kyou: Kyou): void
    (e: 'registered_tag', registred_tag: Tag): void
    (e: 'updated_tag', updated_tag: Tag): void
    (e: 'deleted_tag', deleted_tag: Tag): void
    (e: 'registered_text', registered_text: Text): void
    (e: 'updated_text', updated_text: Text): void
    (e: 'deleted_text', deleted_text: Text): void
    (e: 'registered_notification', registered_notification: Notification): void
    (e: 'updated_notification', updated_notification: Notification): void
    (e: 'deleted_notification', deleted_notification: Notification): void
    (e: 'requested_update_check_kyous', kyou: Array<Kyou>, is_checked: boolean): void
    (e: 'requested_reload_kyou', kyou: Kyou): void
    (e: 'requested_reload_list'): void
    (e: 'requested_close_dialog'): void
    (e: 'clicked_kyou', kyou: Kyou): void
    (e: 'requested_open_rykv_dialog', kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void
}
