'use strict'

import type { ShareMiTaskListInfo } from "@/classes/datas/share-mi-task-list-info"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ShareTaskListLinkViewProps extends GkillPropsBase {
    share_mi_task_list_info: ShareMiTaskListInfo
}
