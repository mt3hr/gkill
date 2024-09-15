'use strict';

import { Lantana } from '@/classes/datas/lantana';
import { GkillAPIRequest } from '../gkill-api-request';


export class AddLantanaRequest extends GkillAPIRequest {


    lantana: Lantana;

    constructor() {
        super()
        this.lantana = new Lantana()
    }


}



