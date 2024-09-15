'use strict';

import { Nlog } from '@/classes/datas/nlog';
import { GkillAPIRequest } from '../gkill-api-request';


export class AddNlogRequest extends GkillAPIRequest {


    nlog: Nlog;

    constructor() {
        super()
        this.nlog = new Nlog()
    }


}



