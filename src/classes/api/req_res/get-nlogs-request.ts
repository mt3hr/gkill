'use strict'

import { FindNlogQuery } from '../find_query/find-nlog-query'
import { GkillAPIRequest } from '../gkill-api-request'

export class GetNlogsRequest extends GkillAPIRequest {

    query: FindNlogQuery

    constructor() {
        super()
        this.query = new FindNlogQuery()
    }

}


