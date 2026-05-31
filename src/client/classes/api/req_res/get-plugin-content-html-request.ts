'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetPluginContentHTMLRequest extends GkillAPIRequest {
    rep_name: string
    kyou_id: string

    constructor() {
        super()
        this.rep_name = ''
        this.kyou_id = ''
    }
}
