'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class RegisterGkillNotificationRequest extends GkillAPIRequest {

    subscription: any

    public_key: string

    constructor() {
        super()
        this.subscription = null
        this.public_key = ""
    }

}


