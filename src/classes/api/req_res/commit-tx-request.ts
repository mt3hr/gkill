'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class CommitTXRequest extends GkillAPIRequest {

    tx_id: string = ""

    constructor() {
        super()
    }

}


