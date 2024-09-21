'use strict'

import { FindReKyouQuery } from '../find_query/find-re-kyou-query'
import { GkillAPIRequest } from '../gkill-api-request'

export class GetReKyousRequest extends GkillAPIRequest {

    query: FindReKyouQuery

    constructor() {
        super()
        this.query = new FindReKyouQuery()
    }

}


