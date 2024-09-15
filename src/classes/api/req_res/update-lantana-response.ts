'use strict';

import { Lantana } from '@/classes/datas/lantana';
import { GkillAPIResponse } from '../gkill-api-response';
import { Kyou } from '@/classes/datas/kyou';


export class UpdateLantanaResponse extends GkillAPIResponse {


    updated_lantana: Lantana;

    updated_lantana_kyou: Kyou;

    constructor() {
        super()
        this.updated_lantana = new Lantana()
        this.updated_lantana_kyou = new Kyou()
    }


}



