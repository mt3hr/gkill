'use strict'

import { FindQueryBase } from './find-query-base'

export class FindKyouQuery extends FindQueryBase {
    reps: Array<string>
    is_image_only: boolean

    async clone(): Promise<FindKyouQuery> {
        throw new Error('Not implemented')
    }

    constructor() {
        super()
        this.reps = new Array<string>()
        this.is_image_only = false
    }
}
