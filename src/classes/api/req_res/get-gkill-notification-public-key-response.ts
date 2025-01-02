'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

export class GetGkillNotificationPublicKeyResponse extends GkillAPIResponse {

    gkill_notification_public_key: string

    constructor() {
        super()
        this.gkill_notification_public_key = ""
    }

}


