// ˅
'use strict';

import { Kyou } from '@/classes/datas/kyou';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class UpdateKyouInfoRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    kyou: Kyou;

    constructor() {
        // ˅
        super()
        this.kyou = new Kyou()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
