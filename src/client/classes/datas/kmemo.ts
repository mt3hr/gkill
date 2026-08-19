'use strict'

import { log_unless_aborted } from '@/classes/abort-error'
import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { GetKmemoRequest } from '../api/req_res/get-kmemo-request'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class Kmemo extends InfoBase {

    content: string

    attached_histories: Array<Kmemo>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetKmemoRequest()
        req.abort_controller = this.abort_controller
        
        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_kmemo(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.kmemo_histories
        return new Array<GkillError>()
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        try {
            return await this.load_attached_histories()
        } catch (err: unknown) {
            // 中断（画面を離れた・後発の検索に差し替わった）は正常なので出さない
            log_unless_aborted(err)
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

    clone(): Kmemo {
        const kmemo = new Kmemo()
        kmemo.is_deleted = this.is_deleted
        kmemo.id = this.id
        kmemo.rep_name = this.rep_name
        kmemo.related_time = this.related_time
        kmemo.data_type = this.data_type
        kmemo.create_time = this.create_time
        kmemo.create_app = this.create_app
        kmemo.create_device = this.create_device
        kmemo.create_user = this.create_user
        kmemo.update_time = this.update_time
        kmemo.update_app = this.update_app
        kmemo.update_user = this.update_user
        kmemo.update_device = this.update_device
        kmemo.content = this.content
        return kmemo
    }

    generate_info_identifier(): InfoIdentifier {
        const info_identifier = new InfoIdentifier()
        info_identifier.id = this.id
        info_identifier.create_time = this.create_time
        info_identifier.update_time = this.update_time
        return info_identifier
    }

    constructor() {
        super()
        this.content = ""
        this.attached_histories = new Array<Kmemo>()
    }

}


