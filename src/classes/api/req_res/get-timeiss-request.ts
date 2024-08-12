// ˅
'use strict';

import { FindTimeIsQuery } from '../find_query/find-time-is-query';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class GetTimeissRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    query: FindTimeIsQuery;

    constructor() {
        // ˅
        super()
        this.query = new FindTimeIsQuery()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
