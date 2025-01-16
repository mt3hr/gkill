'use strict'

import { GkillAPI } from '../api/gkill-api'
import { GkillError } from '../api/gkill-error'
import { GetTimeisRequest } from '../api/req_res/get-timeis-request'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class TimeIs extends InfoBase {

    title: string

    start_time: Date

    end_time: Date | null

    attached_histories: Array<TimeIs>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetTimeisRequest()
        req.abort_controller = this.abort_controller
        
        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_timeis(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.timeis_histories
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

    clone(): TimeIs {
        const timeis = new TimeIs()
        timeis.is_deleted = this.is_deleted
        timeis.id = this.id
        timeis.rep_name = this.rep_name
        timeis.related_time = this.related_time
        timeis.data_type = this.data_type
        timeis.create_time = this.create_time
        timeis.create_app = this.create_app
        timeis.create_device = this.create_device
        timeis.create_user = this.create_user
        timeis.update_time = this.update_time
        timeis.update_app = this.update_app
        timeis.update_user = this.update_user
        timeis.update_device = this.update_device
        timeis.title = this.title
        timeis.start_time = this.start_time
        timeis.end_time = this.end_time
        return timeis
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
        this.title = ""
        this.start_time = new Date(0)
        this.end_time = null
        this.attached_histories = new Array<TimeIs>()
    }

}


