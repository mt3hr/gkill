// ˅
'use strict';

import { Text } from '@/classes/datas/text';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class AddTextResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    added_text: Text;

    constructor() {
        // ˅
        super()
        this.added_text = new Text()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
