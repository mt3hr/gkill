'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { ShareMiTaskListInfo } from "@/classes/datas/share-mi-task-list-info"

export interface ManageShareTaskLinkDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_show_share_task_link_dialog', share_mi_task_list_info: ShareMiTaskListInfo): void
    (e: 'requested_show_confirm_delete_share_task_list_dialog', share_mi_task_list_info: ShareMiTaskListInfo): void
}
