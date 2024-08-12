// ˅
'use strict';

import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class UpdateAccountStatusRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    target_user_id: string;

    enable: boolean;

    constructor() {
        // ˅
        super()
        this.target_user_id = ""
        this.enable = false
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
