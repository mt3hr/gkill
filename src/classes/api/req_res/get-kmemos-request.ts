'use strict'

import { FindKmemoQuery } from '../find_query/find-kmemo-query'
import { GkillAPIRequest } from '../gkill-api-request'

export class GetKmemosRequest extends GkillAPIRequest {

    query: FindKmemoQuery

    constructor() {
        super()
        this.query = new FindKmemoQuery()
    }

}


