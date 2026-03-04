'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class OpenDirectoryRequest extends GkillAPIRequest {
    target_id: string

    constructor() {
        super()
        this.target_id = ''
    }
}


