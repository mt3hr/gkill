'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class RegisterGkillNotificationRequest extends GkillAPIRequest {

    subscription: string

    public_key: string

    constructor() {
        super()
        this.subscription = ""
        this.public_key = ""
    }

}


