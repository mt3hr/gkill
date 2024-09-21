'use strict'

import { FindURLogQuery } from '../find_query/find-ur-log-query'
import { GkillAPIRequest } from '../gkill-api-request'

export class GetURLogsRequest extends GkillAPIRequest {

    query: FindURLogQuery

    constructor() {
        super()
        this.query = new FindURLogQuery()
    }

}


