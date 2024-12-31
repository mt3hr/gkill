'use strict'

import type { ShareMiTaskListInfo } from "@/classes/datas/share-mi-task-list-info"

export interface ConfirmDeleteShareTaskListViewEmits {
    (e: 'requested_delete_share_task_link_info', share_task_link_info: ShareMiTaskListInfo): void
    (e: 'requested_close_dialog'): void
}
