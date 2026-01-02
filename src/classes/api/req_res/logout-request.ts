'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class LogoutRequest extends GkillAPIRequest {

    close_database: boolean

    constructor() {
        super()
        this.close_database = false
    }

}


