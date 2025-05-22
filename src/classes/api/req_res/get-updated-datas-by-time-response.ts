'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

export class GetUpdatedDatasByTimeResponse extends GkillAPIResponse {

    updated_ids: Array<string>

    constructor() {
        super()
        this.updated_ids = new Array<string>()
    }

}


