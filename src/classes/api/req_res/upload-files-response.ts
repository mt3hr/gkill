// ˅
'use strict';

import { Kyou } from '@/classes/datas/kyou';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class UploadFilesResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    uploaded_kyous: Array<Kyou>;

    constructor() {
        // ˅
        super()
        this.uploaded_kyous = new Array<Kyou>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
