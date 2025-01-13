'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { GetLantanaRequest } from '../api/req_res/get-lantana-request'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class Lantana extends InfoBase {

    mood: Number

    attached_histories: Array<Lantana>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetLantanaRequest()
        req.abort_controller = this.abort_controller
        req.session_id = GkillAPI.get_gkill_api().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_lantana(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.lantana_histories
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

    clone(): Lantana {
        const lantana = new Lantana()
        lantana.is_deleted = this.is_deleted
        lantana.id = this.id
        lantana.rep_name = this.rep_name
        lantana.related_time = this.related_time
        lantana.data_type = this.data_type
        lantana.create_time = this.create_time
        lantana.create_app = this.create_app
        lantana.create_device = this.create_device
        lantana.create_user = this.create_user
        lantana.update_time = this.update_time
        lantana.update_app = this.update_app
        lantana.update_user = this.update_user
        lantana.update_device = this.update_device
        lantana.mood = this.mood
        return lantana
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
        this.mood = 0
        this.attached_histories = new Array<Lantana>()
    }

}


