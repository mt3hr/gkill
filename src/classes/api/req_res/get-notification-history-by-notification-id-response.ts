'use strict'

import { Notification } from '@/classes/datas/notification'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetNotificationHistoryByNotificationIDResponse extends GkillAPIResponse {

    notification_histories: Array<Notification>

    constructor() {
        super()
        this.notification_histories = new Array<Notification>()
    }

}


