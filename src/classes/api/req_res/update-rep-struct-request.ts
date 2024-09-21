'use strict'

import { RepStruct } from '@/classes/datas/config/rep-struct'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateRepStructRequest extends GkillAPIRequest {

    rep_struct: Array<RepStruct>

    constructor() {
        super()
        this.rep_struct = new Array<RepStruct>()
    }

}


