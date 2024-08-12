// ˅
'use strict';

import { Kmemo } from '@/classes/datas/kmemo';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class GetKmemoResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    kmemo_histories: Array<Kmemo>;

    constructor() {
        // ˅
        super()
        this.kmemo_histories = new Array<Kmemo>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
