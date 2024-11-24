'use strict'

import { FindQueryBase } from './find-query-base'
import { FindKyouQuery } from './find-kyou-query'

export class FindReKyouQuery extends FindQueryBase {

    clone(): FindReKyouQuery {
        throw new Error('Not implemented')
    }

    async generate_find_kyou_query(): Promise<FindKyouQuery> {
        throw new Error('Not implemented')
    }

    constructor() {
        super()
    }

}


