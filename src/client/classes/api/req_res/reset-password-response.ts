'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

export class ResetPasswordResponse extends GkillAPIResponse {

    password_reset_path_without_host: string

    constructor() {
        super()
        this.password_reset_path_without_host = ""

    }

}


