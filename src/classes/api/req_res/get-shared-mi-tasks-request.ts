'use strict'

import { GkillAPIRequest } from "../gkill-api-request"

export class GetSharedMiTasksRequest extends GkillAPIRequest {

    shared_id: string

    constructor() {
        super()
        this.shared_id = ""
    }

}


