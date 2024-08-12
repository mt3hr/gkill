// ˅
'use strict';

import { GPSLog } from '@/classes/datas/gps-log';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class GetGPSLogResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    gps_logs: Array<GPSLog>;

    constructor() {
        // ˅
        super()
        this.gps_logs = new Array<GPSLog>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
