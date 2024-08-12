// ˅
'use strict';

import { Lantana } from '@/classes/datas/lantana';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class GetLantanasResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    lantanas: Array<Lantana>;

    constructor() {
        // ˅
        super()
        this.lantanas = new Array<Lantana>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
