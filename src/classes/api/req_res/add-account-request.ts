'use strict';

import { Account } from '@/classes/datas/config/account';
import { GkillAPIRequest } from '../gkill-api-request';


export class AddAccountRequest extends GkillAPIRequest {


    account_info: Account;

    do_initialize: boolean;

    constructor() {
        super()
        this.account_info = new Account()
        this.do_initialize = false
    }


}



