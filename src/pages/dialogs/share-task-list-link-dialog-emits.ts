'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { ShareMiTaskListInfo } from "@/classes/datas/share-mi-task-list-info"

export interface ShareTaskListLinkDialogEmits {
    (e: 'updated_share_mi_task_list_info', share_mi_task_list_info: ShareMiTaskListInfo): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}
