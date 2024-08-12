// ˅
'use strict';

import { Repository } from '@/classes/datas/config/repository';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class UpdateUserRepsRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    target_user_id: string;

    updated_reps: Array<Repository>;

    constructor() {
        // ˅
        super()
        this.target_user_id = ""
        this.updated_reps = new Array<Repository>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
