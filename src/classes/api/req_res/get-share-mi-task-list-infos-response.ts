'use strict';

import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info';
import { GkillAPIResponse } from '../gkill-api-response';


export class GetShareMiTaskListInfosResponse extends GkillAPIResponse {


    share_mi_task_list_infos: Array<ShareMiTaskListInfo>;

    constructor() {
        super()
        this.share_mi_task_list_infos = new Array<ShareMiTaskListInfo>()
    }


}



