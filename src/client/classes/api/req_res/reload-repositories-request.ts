'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class ReloadRepositoriesRequest extends GkillAPIRequest {
    clear_thumb_cache: boolean
    clear_video_cache: boolean
    clear_zip_cache: boolean

    constructor() {
        super()
        this.clear_thumb_cache = false
        this.clear_video_cache = false
        this.clear_zip_cache = false
    }
}


