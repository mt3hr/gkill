'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

export class GetGkillInfoResponse extends GkillAPIResponse {

    user_id: string

    device: string

    user_is_admin: boolean

    cache_clear_count_limit: number

    constructor() {
        super()
        this.user_id = ""
        this.device = ""
        this.user_is_admin = false
        this.cache_clear_count_limit = 1001
    }

}


