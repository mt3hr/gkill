'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetMiReKyousByTargetIDRequest extends GkillAPIRequest {

    target_id: string


    constructor() {
        super()
        this.target_id = ""
    }

}
