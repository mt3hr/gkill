'use strict';

import { Mi } from '@/classes/datas/mi';
import { GkillAPIRequest } from '../gkill-api-request';


export class AddMiRequest extends GkillAPIRequest {


    mi: Mi;

    constructor() {
        super()
        this.mi = new Mi()
    }


}



