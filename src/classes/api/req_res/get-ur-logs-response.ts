// ˅
'use strict';

import { URLog } from '@/classes/datas/ur-log';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class GetURLogsResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    urlogs: Array<URLog>;

    constructor() {
        // ˅
        super()
        this.urlogs = new Array<URLog>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
