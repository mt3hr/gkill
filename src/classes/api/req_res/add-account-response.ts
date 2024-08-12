// ˅
'use strict';

import { Account } from '@/classes/datas/config/account';
import { GkillAPIResponse } from '../gkill-api-response';

// ˄

export class AddAccountResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    added_account_info: Account;

    constructor() {
        // ˅
        super()
        this.added_account_info = new Account()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
