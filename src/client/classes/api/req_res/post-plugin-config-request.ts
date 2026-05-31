'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class PostPluginConfigRequest extends GkillAPIRequest {
    rep_name: string
    form_data: Record<string, string>

    constructor() {
        super()
        this.rep_name = ''
        this.form_data = {}
    }
}
