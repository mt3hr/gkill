'use strict'

import { Tag } from '@/classes/datas/tag'
import { GkillAPIResponse } from '../gkill-api-response'

export class UpdateTagResponse extends GkillAPIResponse {

    updated_tag: Tag

    constructor() {
        super()
        this.updated_tag = new Tag()
    }

}


