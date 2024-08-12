// ˅
'use strict';

import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class ResetPasswordRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    target_user_id: string;

    constructor() {
        // ˅
        super()
        this.target_user_id = "";
        
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
