'use strict';

import { ReKyou } from '@/classes/datas/re-kyou';
import { GkillAPIResponse } from '../gkill-api-response';


export class GetReKyousResponse extends GkillAPIResponse {


    rekyous: Array<ReKyou>;

    constructor() {
        super()
        this.rekyous = new Array<ReKyou>
    }


}



