'use strict';

import type { ShareMiTaskListInfo } from "@/classes/datas/share-mi-task-list-info";

export interface ConfirmDeleteShareTaskLinkDialogEmits {
    (e: 'requested_close_dialog'): void
    (e: 'deleted_share_mi_task_list_info', share_mi_task_list_info: ShareMiTaskListInfo): void
}