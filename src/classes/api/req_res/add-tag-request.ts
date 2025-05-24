'use strict'

import { Tag } from '@/classes/datas/tag'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddTagRequest extends GkillAPIRequest {

    tag: Tag

    tx_id: string | null = null

    constructor() {
        super()
        this.tag = new Tag()
    }

}


