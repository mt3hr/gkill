'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { GetNotificationHistoryByNotificationIDRequest } from '../api/req_res/get-notification-history-by-notification-id-request copy'
import { InfoIdentifier } from './info-identifier'
import { MetaInfoBase } from './meta-info-base'

export class Notification extends MetaInfoBase {

    // related_time は使わない

    content: string

    is_notificated: boolean

    notification_time: Date

    attached_histories: Array<Notification>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetNotificationHistoryByNotificationIDRequest()

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_notification_history_by_notification_id(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.notification_histories
        return new Array<GkillError>()
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        try {
            return this.load_attached_histories()
        } catch (err: any) {
            // abortは握りつぶす
            if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
                // abort以外はエラー出力する
                console.error(err)
            }
            return []
        }
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        this.attached_histories = []
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        return this.clear_attached_histories()
    }

    clone(): Notification {
        const notification = new Notification()
        notification.is_deleted = this.is_deleted
        notification.id = this.id
        notification.is_notificated = this.is_notificated
        notification.target_id = this.target_id
        notification.related_time = this.related_time
        notification.create_time = this.create_time
        notification.create_app = this.create_app
        notification.create_device = this.create_device
        notification.create_user = this.create_user
        notification.update_time = this.update_time
        notification.update_app = this.update_app
        notification.update_device = this.update_device
        notification.update_user = this.update_user
        notification.notification_time = this.notification_time
        notification.content = this.content
        return notification
    }

    generate_info_identifer(): InfoIdentifier {
        const info_identifer = new InfoIdentifier()
        info_identifer.id = this.id
        info_identifer.create_time = this.create_time
        info_identifer.update_time = this.update_time
        return info_identifer
    }

    constructor() {
        super()
        this.notification_time = new Date(0)
        this.content = ""
        this.attached_histories = new Array<Notification>()
        this.is_notificated = false
    }

}


