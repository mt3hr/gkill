'use strict';

import { Kmemo } from '@/classes/datas/kmemo';
import { GkillAPIResponse } from '../gkill-api-response';
import { Kyou } from '@/classes/datas/kyou';


export class AddKmemoResponse extends GkillAPIResponse {


    added_kmemo: Kmemo;

    added_kmemo_kyou: Kyou;

    constructor() {
        super()
        this.added_kmemo = new Kmemo()
        this.added_kmemo_kyou = new Kyou()
    }


}



