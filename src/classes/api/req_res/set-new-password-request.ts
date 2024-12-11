'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class SetNewPasswordRequest extends GkillAPIRequest {

    user_id: string

    reset_token: string

    new_password_sha256: string

    constructor() {
        super()
        this.user_id = ""
        this.reset_token = ""
        this.new_password_sha256 = ""

    }

}


