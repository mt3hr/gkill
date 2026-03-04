'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetGPSLogRequest extends GkillAPIRequest {

    start_date: Date

    end_date: Date

    constructor() {
        super()
        this.start_date = new Date(0)
        this.end_date = new Date(0)
    }

}


