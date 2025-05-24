'use strict'

import { RepTypeStruct } from '@/classes/datas/config/rep-type-struct'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateRepTypeStructRequest extends GkillAPIRequest {

    rep_type_struct: Array<RepTypeStruct>

    tx_id: string | null = null

    constructor() {
        super()
        this.rep_type_struct = new Array<RepTypeStruct>()
    }

}


