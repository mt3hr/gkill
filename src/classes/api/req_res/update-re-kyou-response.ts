// ˅
'use strict';

import { ReKyou } from '@/classes/datas/re-kyou';
import { GkillAPIResponse } from '../gkill-api-response';
import { Kyou } from '@/classes/datas/kyou';

// ˄

export class UpdateReKyouResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    updated_rekyou: ReKyou;

    updated_rekyou_kyou: Kyou;

    constructor() {
        // ˅
        super()
        this.updated_rekyou = new ReKyou()
        this.updated_rekyou_kyou = new Kyou()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
