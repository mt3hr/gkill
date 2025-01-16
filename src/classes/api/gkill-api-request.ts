'use strict'

import { GkillAPI } from "./gkill-api"

export class GkillAPIRequest {

    abort_controller: AbortController

    session_id: string

    constructor() {
        this.session_id = GkillAPI.get_gkill_api().get_session_id()
        this.abort_controller = new AbortController()
    }

}


