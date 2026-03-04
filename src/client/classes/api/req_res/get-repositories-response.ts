'use strict'

import { GkillAPIResponse } from '../gkill-api-response'
import { Repository } from '@/classes/datas/config/repository'

export class GetRepositoriesResponse extends GkillAPIResponse {

    repositories: Array<Repository>

    constructor() {
        super()
        this.repositories = new Array<Repository>
    }

}


