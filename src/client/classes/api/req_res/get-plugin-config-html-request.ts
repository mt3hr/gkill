'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetPluginConfigHTMLRequest extends GkillAPIRequest {
    rep_name: string

    // target_user_id が空なら自分自身。どちらにしても管理者だけが通る
    target_user_id: string

    constructor() {
        super()
        this.rep_name = ''
        this.target_user_id = ""
    }
}
