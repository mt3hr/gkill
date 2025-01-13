'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { GetURLogRequest } from '../api/req_res/get-ur-log-request'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class URLog extends InfoBase {

    url: string

    title: string

    description: string

    favicon_image: string

    thumbnail_image: string

    attached_histories: Array<URLog>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetURLogRequest()
        req.abort_controller = this.abort_controller
        req.session_id = GkillAPI.get_gkill_api().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_urlog(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.urlog_histories
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
        let errors = new Array<GkillError>()
        errors = errors.concat(await this.clear_attached_tags())
        errors = errors.concat(await this.clear_attached_texts())
        errors = errors.concat(await this.clear_attached_notifications())
        errors = errors.concat(await this.clear_attached_timeis())
        errors = errors.concat(await this.clear_attached_histories())
        return errors
    }

    clone(): URLog {
        const urlog = new URLog()
        urlog.is_deleted = this.is_deleted
        urlog.id = this.id
        urlog.rep_name = this.rep_name
        urlog.related_time = this.related_time
        urlog.data_type = this.data_type
        urlog.create_time = this.create_time
        urlog.create_app = this.create_app
        urlog.create_device = this.create_device
        urlog.create_user = this.create_user
        urlog.update_time = this.update_time
        urlog.update_app = this.update_app
        urlog.update_user = this.update_user
        urlog.update_device = this.update_device
        urlog.url = this.url
        urlog.title = this.title
        urlog.description = this.description
        urlog.favicon_image = this.favicon_image
        urlog.thumbnail_image = this.thumbnail_image
        return urlog
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
        this.url = ""
        this.title = ""
        this.description = ""
        this.favicon_image = ""
        this.thumbnail_image = ""
        this.attached_histories = new Array<URLog>()
    }

}


