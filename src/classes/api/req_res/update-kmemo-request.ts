// ˅
'use strict';

import { Kmemo } from '@/classes/datas/kmemo';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class UpdateKmemoRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    kmemo: Kmemo;

    constructor() {
        // ˅
        super()
        this.kmemo = new Kmemo()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
