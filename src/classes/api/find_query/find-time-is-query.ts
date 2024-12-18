'use strict'

import { FindQueryBase } from './find-query-base'
import { FindKyouQuery } from './find-kyou-query'

export class FindTimeIsQuery extends FindQueryBase {

    plaing_only: boolean

    use_plaing: boolean

    plaing_time: Date | null

    include_end_timeis: boolean

    clone(): FindTimeIsQuery {
        throw new Error('Not implemented')
    }

    async generate_find_kyou_query(): Promise<FindKyouQuery> {
        throw new Error('Not implemented')
    }

    constructor() {
        super()
        this.plaing_only = false
        this.use_plaing = false
        this.plaing_time = null
        this.include_end_timeis = false
    }

}


