// ˅
'use strict';

import { Tag } from '@/classes/datas/tag';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class UpdateTagRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    tag: Tag;

    constructor() {
        // ˅
        super()
        this.tag = new Tag()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄