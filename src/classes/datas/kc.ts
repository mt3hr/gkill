'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { GetKCRequest } from '../api/req_res/get-kc-request'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class KC extends InfoBase {

    title: string

    num_value: number

    attached_histories: Array<KC>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetKCRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_kc(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.kc_histories
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

    clone(): KC {
        const kc = new KC()
        kc.is_deleted = this.is_deleted
        kc.id = this.id
        kc.rep_name = this.rep_name
        kc.related_time = this.related_time
        kc.data_type = this.data_type
        kc.create_time = this.create_time
        kc.create_app = this.create_app
        kc.create_device = this.create_device
        kc.create_user = this.create_user
        kc.update_time = this.update_time
        kc.update_app = this.update_app
        kc.update_user = this.update_user
        kc.update_device = this.update_device
        kc.title = this.title
        kc.num_value = this.num_value
        return kc
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
        this.num_value = 0
        this.attached_histories = new Array<KC>()
    }

}


