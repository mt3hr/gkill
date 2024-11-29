'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

export class GetAllRepNamesResponse extends GkillAPIResponse {

    rep_names: Array<string>

    constructor() {
        super()
        this.rep_names = new Array<string>()
    }

}


