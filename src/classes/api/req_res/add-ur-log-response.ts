'use strict';

import { URLog } from '@/classes/datas/ur-log';
import { GkillAPIResponse } from '../gkill-api-response';
import { Kyou } from '@/classes/datas/kyou';


export class AddURLogResponse extends GkillAPIResponse {


    added_urlog: URLog;

    added_urlog_kyou: Kyou;

    constructor() {
        super()
        this.added_urlog = new URLog()
        this.added_urlog_kyou = new Kyou()
    }


}



