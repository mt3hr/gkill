'use strict'

import { TagStruct } from '@/classes/datas/config/tag-struct'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateTagStructRequest extends GkillAPIRequest {

    tag_struct: Array<TagStruct>

    constructor() {
        super()
        this.tag_struct = new Array<TagStruct>()
    }

}


