'use strict'

import { FindQueryBase } from './find-query-base'
import { FindKyouQuery } from './find-kyou-query'

export class FindURLogQuery extends FindQueryBase {

    clone(): FindURLogQuery {
        throw new Error('Not implemented')
    }

    async generate_find_kyou_query(): Promise<FindKyouQuery> {
        throw new Error('Not implemented')
    }

    constructor() {
        super()
    }

}


