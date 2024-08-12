// ˅
'use strict';

import { Lantana } from '@/classes/datas/lantana';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class UpdateLantanaRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    lantana: Lantana;

    constructor() {
        // ˅
        super()
        this.lantana = new Lantana()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
