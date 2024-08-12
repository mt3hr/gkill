// ˅
'use strict';

import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class GetGkillInfoResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    user_id: string;

    device: string;

    constructor() {
        // ˅
        super()
        this.user_id = ""
        this.device = ""
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
