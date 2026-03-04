'use strict'

import { GkillAPIRequest } from "../gkill-api-request"

export class GetSharedKyousRequest extends GkillAPIRequest {

    shared_id: string

    constructor() {
        super()
        this.shared_id = ""
    }

}


