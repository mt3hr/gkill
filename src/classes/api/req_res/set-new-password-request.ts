'use strict';

import { GkillAPIRequest } from '../gkill-api-request';


export class SetNewPasswordRequest extends GkillAPIRequest {


    user_id: string;

    new_password: string;

    constructor() {
        super()
        this.user_id = "";
        this.new_password = "";

    }


}



