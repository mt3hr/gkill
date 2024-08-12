// ˅
'use strict';

import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class LoginResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    session_id: string;

    constructor() {
        // ˅
        super()
        this.session_id = "";
        
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
