// ˅
'use strict';

import { Kmemo } from '@/classes/datas/kmemo';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class GetKmemosResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    kmemos: Array<Kmemo>;

    constructor() {
        // ˅
        super()
        this.kmemos = new Array<Kmemo>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
