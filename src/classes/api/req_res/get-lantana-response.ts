'use strict';

import { Lantana } from '@/classes/datas/lantana';
import { GkillAPIResponse } from '../gkill-api-response';


export class GetLantanaResponse extends GkillAPIResponse {


    lantana_histories: Array<Lantana>;

    constructor() {
        super()
        this.lantana_histories = new Array<Lantana>()
    }


}



