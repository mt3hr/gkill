// ˅
'use strict';

import { FileData } from '../file-data';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class UploadGPSLogFilesRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    gpslog_files: Array<FileData>;

    constructor() {
        // ˅
        super()
        this.gpslog_files = new Array<FileData>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
