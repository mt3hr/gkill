'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"
import type { Text } from "@/classes/datas/text"
import type { Notification } from "@/classes/datas/notification"
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"

export interface KyouListViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'registered_kyou', kyou: Kyou): void
    (e: 'updated_kyou', kyou: Kyou): void
    (e: 'deleted_kyou', kyou: Kyou): void
    (e: 'registered_tag', tag: Tag): void
    (e: 'updated_tag', tag: Tag): void
    (e: 'deleted_tag', tag: Tag): void
    (e: 'registered_text', text: Text): void
    (e: 'updated_text', text: Text): void
    (e: 'deleted_text', text: Text): void
    (e: 'registered_notification', notification: Notification): void
    (e: 'updated_notification', notification: Notification): void
    (e: 'deleted_notification', notification: Notification): void
    (e: 'requested_update_check_kyous', kyou: Array<Kyou>, is_checked: boolean): void
    (e: 'requested_reload_kyou', kyou: Kyou): void
    (e: 'requested_reload_list'): void
    (e: 'requested_close_dialog'): void
    (e: 'focused_kyou', kyou: Kyou): void
    (e: 'clicked_kyou', kyou: Kyou): void
    (e: 'requested_search'): void
    (e: 'requested_close_column'): void
    (e: 'requested_change_is_image_only_view', is_image_only_view: boolean): void
    (e: 'requested_change_focus_kyou', is_focus_kyou: boolean): void
    (e: 'scroll_list', top: number): void
    (e: 'clicked_list_view'): void
    (e: 'requested_open_rykv_dialog', kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void
}
