'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetTextHistoryByTextIDRequest extends GkillAPIRequest {

    id: string

    constructor() {
        super()
        this.id = ""
    }

}


