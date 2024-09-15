'use strict';

import { Kmemo } from '@/classes/datas/kmemo';
import { GkillAPIRequest } from '../gkill-api-request';


export class AddKmemoRequest extends GkillAPIRequest {


    kmemo: Kmemo;

    constructor() {
        super()
        this.kmemo = new Kmemo()
    }


}



