'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class ReloadRepositoriesRequest extends GkillAPIRequest {
    clear_thumb_cache: boolean

    constructor() {
        super()
        this.clear_thumb_cache = false
    }
}


