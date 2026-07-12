'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

export class GetIDFKyouByRelativePathResponse extends GkillAPIResponse {

    // 見つからなかった場合は空文字
    kyou_id: string

    constructor() {
        super()
        this.kyou_id = ''
    }

}
