'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetKmemoRequest extends GkillAPIRequest {

    update_time: Date | null
    id: string

    constructor() {
        super()
        this.id = ""
        this.update_time = null
    }

}


