'use strict'

import type { ShareMiTaskListInfo } from "@/classes/datas/share-mi-task-list-info"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ManageShareTaskListViewProps extends GkillPropsBase {
    share_mi_task_list_infos: Array<ShareMiTaskListInfo>
}
