'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { GetNlogRequest } from '../api/req_res/get-nlog-request'
import { GetNlogsRequest } from '../api/req_res/get-nlogs-request'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class Nlog extends InfoBase {

    shop: string

    title: string

    amount: Number

    attached_histories: Array<Nlog>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetNlogRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_nlog(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.nlog_histories
        return new Array<GkillError>()
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        return this.load_attached_histories()
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        this.attached_histories = []
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        errors = errors.concat(await this.clear_attached_tags())
        errors = errors.concat(await this.clear_attached_texts())
        errors = errors.concat(await this.clear_attached_timeis())
        errors = errors.concat(await this.clear_attached_histories())
        return errors
    }


    clone(): Nlog {
        const nlog = new Nlog()
        nlog.is_deleted = this.is_deleted
        nlog.id = this.id
        nlog.rep_name = this.rep_name
        nlog.related_time = this.related_time
        nlog.data_type = this.data_type
        nlog.create_time = this.create_time
        nlog.create_app = this.create_app
        nlog.create_device = this.create_device
        nlog.create_user = this.create_user
        nlog.update_time = this.update_time
        nlog.update_app = this.update_app
        nlog.update_user = this.update_user
        nlog.update_device = this.update_device
        nlog.is_checked = this.is_checked
        nlog.shop = this.shop
        nlog.title = this.title
        nlog.amount = this.amount
        return nlog
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
        this.shop = ""
        this.title = ""
        this.amount = 0
        this.attached_histories = new Array<Nlog>()
    }

}


