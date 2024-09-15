'use strict';

import { Nlog } from '@/classes/datas/nlog';
import { GkillAPIResponse } from '../gkill-api-response';
import { Kyou } from '@/classes/datas/kyou';


export class UpdateNlogResponse extends GkillAPIResponse {


    updated_nlog: Nlog;

    updated_nlog_kyou: Kyou;

    constructor() {
        super()
        this.updated_nlog = new Nlog()
        this.updated_nlog_kyou = new Kyou()
    }


}



