'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetUpdatedDatasByTimeRequest extends GkillAPIRequest {

    last_updated_time: Date

    constructor() {
        super()
        this.last_updated_time = new Date(0)
    }

}


