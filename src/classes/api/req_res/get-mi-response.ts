'use strict';

import { Mi } from '@/classes/datas/mi';
import { GkillAPIResponse } from '../gkill-api-response';


export class GetMiResponse extends GkillAPIResponse {


    mi_histories: Array<Mi>;

    constructor() {
        super()
        this.mi_histories = new Array<Mi>()
    }


}



