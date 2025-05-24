'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class DiscardTXRequest extends GkillAPIRequest {

    tx_id: string = ""

    constructor() {
        super()
    }

}


