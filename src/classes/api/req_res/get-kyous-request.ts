'use strict'

import { FindKyouQuery } from '../find_query/find-kyou-query'
import { GkillAPIRequest } from '../gkill-api-request'

export class GetKyousRequest extends GkillAPIRequest {

    query: FindKyouQuery

    constructor() {
        super()
        this.query = new FindKyouQuery()
    }

}


