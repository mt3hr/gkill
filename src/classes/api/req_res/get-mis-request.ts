// ˅
'use strict';

import { FindMiQuery } from '../find_query/find-mi-query';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class GetMisRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    query: FindMiQuery;

    constructor() {
        // ˅
        super()
        this.query = new FindMiQuery()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
