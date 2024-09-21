'use strict';

import { FileData } from '../file-data';
import { GkillAPIRequest } from '../gkill-api-request';
import { FileUploadConflictBehavior } from './file-upload-conflict-behavior';


export class UploadGPSLogFilesRequest extends GkillAPIRequest {


    gpslog_files: Array<FileData>;

    target_rep_name: string;

    conflict_behavior: FileUploadConflictBehavior;

    constructor() {
        super()
        this.gpslog_files = new Array<FileData>()
        this.target_rep_name = ""
        this.conflict_behavior = FileUploadConflictBehavior.merge
    }


}



