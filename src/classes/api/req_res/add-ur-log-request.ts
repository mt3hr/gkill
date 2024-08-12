// ˅
'use strict';

import { URLog } from '@/classes/datas/ur-log';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class AddURLogRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    urlog: URLog;

    constructor() {
        // ˅
        super()
        this.urlog = new URLog()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
