'use strict'

import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddShareMiTaskListInfoRequest extends GkillAPIRequest {

    share_mi_task_list_info: ShareMiTaskListInfo

    constructor() {
        super()
        this.share_mi_task_list_info = new ShareMiTaskListInfo()
    }

}


