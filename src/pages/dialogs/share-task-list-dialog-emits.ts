'use strict';

import type { ShareMiTaskListInfo } from "@/classes/datas/share-mi-task-list-info";

export interface ShareTaskListDialogEmits {
    (e: 'requested_close_dialog'): void
    (e: 'regestered_share_mi_task_list_info', share_mi_task_list_info: ShareMiTaskListInfo): void
    (e: 'requested_show_share_task_link_dialog', share_mi_task_list_info: ShareMiTaskListInfo): void
}