'use strict'

import type { GkillError } from '../api/gkill-error'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class IDFKyou extends InfoBase {

    file_name: string

    file_url: string

    is_image: boolean

    is_video: boolean

    is_audio: boolean

    attached_histories: Array<IDFKyou>

    async load_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
        /*
        const req = new GetIDFKyouRequest()
        req.abort_controller = this.abort_controller
        
        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_idf_kyou(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.kmemo_histories
        return new Array<GkillError>()
        */
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



    clone(): IDFKyou {
        const idf_kyou = new IDFKyou()
        idf_kyou.is_deleted = this.is_deleted
        idf_kyou.id = this.id
        idf_kyou.rep_name = this.rep_name
        idf_kyou.related_time = this.related_time
        idf_kyou.data_type = this.data_type
        idf_kyou.create_time = this.create_time
        idf_kyou.create_app = this.create_app
        idf_kyou.create_device = this.create_device
        idf_kyou.create_user = this.create_user
        idf_kyou.update_time = this.update_time
        idf_kyou.update_app = this.update_app
        idf_kyou.update_user = this.update_user
        idf_kyou.update_device = this.update_device
        idf_kyou.file_name = this.file_name
        idf_kyou.file_url = this.file_url
        idf_kyou.is_image = this.is_image
        idf_kyou.is_video = this.is_video
        idf_kyou.is_audio = this.is_audio
        return idf_kyou
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
        this.file_name = ""
        this.file_url = ""
        this.is_image = false
        this.is_video = false
        this.is_audio = false
        this.attached_histories = new Array<IDFKyou>()
    }

}


