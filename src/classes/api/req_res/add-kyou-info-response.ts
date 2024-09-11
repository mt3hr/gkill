// ˅
'use strict';

import { GkillAPIResponse } from '../gkill-api-response';
import { IDFKyou } from '@/classes/datas/idf-kyou';

// ˄

export class AddKyouInfoResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    added_kyou: IDFKyou;

    constructor() {
        // ˅
        super()
        this.added_kyou = new IDFKyou()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
