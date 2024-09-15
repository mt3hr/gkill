'use strict';

import { URLog } from '@/classes/datas/ur-log';
import { GkillAPIResponse } from '../gkill-api-response';
import { Kyou } from '@/classes/datas/kyou';


export class UpdateURLogResponse extends GkillAPIResponse {


    updated_urlog: URLog;

    updated_urlog_kyou: Kyou;

    constructor() {
        super()
        this.updated_urlog = new URLog()
        this.updated_urlog_kyou = new Kyou()
    }


}



