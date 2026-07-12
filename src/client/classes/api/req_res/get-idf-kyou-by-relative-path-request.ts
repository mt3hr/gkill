'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetIDFKyouByRelativePathRequest extends GkillAPIRequest {

    target_id: string

    relative_path: string

    constructor() {
        super()
        this.target_id = ''
        this.relative_path = ''
    }

}
