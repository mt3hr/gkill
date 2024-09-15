'use strict';

import { TimeIs } from '@/classes/datas/time-is';
import { GkillAPIRequest } from '../gkill-api-request';


export class AddTimeisRequest extends GkillAPIRequest {


    timeis: TimeIs;

    constructor() {
        super()
        this.timeis = new TimeIs()
    }


}



