'use strict'

import { Notification } from '@/classes/datas/notification'
import { GkillAPIResponse } from '../gkill-api-response'

export class UpdateNotificationResponse extends GkillAPIResponse {

    updated_notification: Notification

    constructor() {
        super()
        this.updated_notification = new Notification()
    }

}


