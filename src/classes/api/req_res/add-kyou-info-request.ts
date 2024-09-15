'use strict';

import { IDFKyou } from '@/classes/datas/idf-kyou';
import { GkillAPIRequest } from '../gkill-api-request';


export class AddKyouInfoRequest extends GkillAPIRequest {


    kyou: IDFKyou;

    constructor() {
        super()
        this.kyou = new IDFKyou()
    }


}



