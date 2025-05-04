'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateDnoteJSONDataRequest extends GkillAPIRequest {

    dnote_json_data: any

    constructor() {
        super()
        this.dnote_json_data = {}
    }

}


