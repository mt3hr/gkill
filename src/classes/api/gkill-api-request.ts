'use strict'

import { GkillAPI } from "./gkill-api"

export class GkillAPIRequest {

    abort_controller: AbortController

    session_id: string

    force_reget: boolean

    constructor() {
        this.session_id = GkillAPI.get_gkill_api().get_session_id()
        this.abort_controller = new AbortController()
        this.force_reget = false
    }

}


