'use strict';

import { FileData } from '../file-data';
import { GkillAPIRequest } from '../gkill-api-request';
import { FileUploadConflictBehavior } from './file-upload-conflict-behavior';


export class UploadFilesRequest extends GkillAPIRequest {


    files: Array<FileData>;

    target_rep_name: string;

    conflict_behavior: FileUploadConflictBehavior;

    constructor() {
        super()
        this.files = new Array<FileData>()
        this.target_rep_name = ""
        this.conflict_behavior = FileUploadConflictBehavior.rename
    }


}



