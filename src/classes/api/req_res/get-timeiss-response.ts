'use strict';

import { TimeIs } from '@/classes/datas/time-is';
import { GkillAPIResponse } from '../gkill-api-response';


export class GetTimeissResponse extends GkillAPIResponse {


    timeiss: Array<TimeIs>;

    constructor() {
        super()
        this.timeiss = new Array<TimeIs>()
    }


}



