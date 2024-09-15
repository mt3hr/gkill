'use strict';

import type { ShareMiTaskListInfo } from "@/classes/datas/share-mi-task-list-info";
import type { GkillPropsBase } from "../views/gkill-props-base";

export interface ShareTaskListLinkDialogProps extends GkillPropsBase {
    is_show: boolean;
    share_mi_task_list_info: ShareMiTaskListInfo;
}