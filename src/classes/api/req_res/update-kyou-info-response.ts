// ˅
'use strict';

import { Kyou } from '@/classes/datas/kyou';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class UpdateKyouInfoResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    updated_kyou: Kyou;

    constructor() {
        // ˅
        super()
        this.updated_kyou = new Kyou()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
