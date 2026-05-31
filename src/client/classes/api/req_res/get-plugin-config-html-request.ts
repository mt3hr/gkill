'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetPluginConfigHTMLRequest extends GkillAPIRequest {
    rep_name: string

    constructor() {
        super()
        this.rep_name = ''
    }
}
