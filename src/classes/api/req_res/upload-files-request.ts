// ˅
'use strict';

import { FileData } from '../file-data';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class UploadFilesRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    files: Array<FileData>;

    target_rep_name: string;

    conflict_behavior: FileUploadConflictBehavior;

    constructor() {
        // ˅
        super()
        this.files = new Array<FileData>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
