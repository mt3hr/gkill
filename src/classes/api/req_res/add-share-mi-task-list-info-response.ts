'use strict';

import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info';
import { GkillAPIResponse } from '../gkill-api-response';


export class AddShareMiTaskListInfoResponse extends GkillAPIResponse {


    share_mi_task_list_info: ShareMiTaskListInfo;

    constructor() {
        super()
        this.share_mi_task_list_info = new ShareMiTaskListInfo()
    }


}



