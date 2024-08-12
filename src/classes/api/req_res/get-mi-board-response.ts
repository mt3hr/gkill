// ˅
'use strict';

import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class GetMiBoardResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    boards: Array<string>;

    constructor() {
        // ˅
        super()
        this.boards = new Array<string>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
