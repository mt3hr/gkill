// ˅
'use strict';

import { TimeIs } from '@/classes/datas/time-is';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class GetPlaingTimeisResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    plaing_timeiss: Array<TimeIs>;

    constructor() {
        // ˅
        super()
        this.plaing_timeiss = new Array<TimeIs>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
