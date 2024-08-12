// ˅
'use strict';

import { Kyou } from '@/classes/datas/kyou';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class AddKyouInfoResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    added_kyou: Kyou;

    constructor() {
        // ˅
        super()
        this.added_kyou = new Kyou()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
