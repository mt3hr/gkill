'use strict'

import type { Tag } from '@/classes/datas/tag'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetTagHistoryByTagIDResponse extends GkillAPIResponse {

    tag_histories: Array<Tag>

    constructor() {
        super()
        this.tag_histories = new Array<Tag>()
    }

}


