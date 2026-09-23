'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class DownloadSkillRequest extends GkillAPIRequest {
    name: string

    constructor() {
        super()
        this.name = ""
    }
}
