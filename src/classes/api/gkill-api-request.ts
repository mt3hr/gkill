'use strict'

export class GkillAPIRequest {

    abort_controller: AbortController | null

    session_id: string

    constructor() {
        this.session_id = ""
        this.abort_controller = null
    }

}


