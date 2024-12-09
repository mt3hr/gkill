'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetNlogRequest extends GkillAPIRequest {

    id: string
    update_time: Date | null

    constructor() {
        super()
        this.id = ""
        this.update_time = null
    }

}


