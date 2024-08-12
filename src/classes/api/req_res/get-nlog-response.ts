// ˅
'use strict';

import { Nlog } from '@/classes/datas/nlog';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class GetNlogResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    nlog_histories: Array<Nlog>;

    constructor() {
        // ˅
        super()
        this.nlog_histories = new Array<Nlog>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
