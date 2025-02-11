'use strict'

import { GkillAPI } from '../api/gkill-api'
import { GkillError } from '../api/gkill-error'
import { GkillErrorCodes } from '../api/message/gkill_error'
import { GetKyouRequest } from '../api/req_res/get-kyou-request'
import { GetReKyouRequest } from '../api/req_res/get-re-kyou-request'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'
import { Kyou } from './kyou'

export class ReKyou extends InfoBase {

    target_id: string

    attached_kyou: Kyou | null

    attached_histories: Array<ReKyou>

    async load_attached_kyou(): Promise<Array<GkillError>> {
        const req = new GetKyouRequest()
        req.abort_controller = this.abort_controller

        req.id = this.target_id
        const res = await GkillAPI.get_gkill_api().get_kyou(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        if (!res.kyou_histories || res.kyou_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_rekyou_target
            error.error_message = "ReKyou対象の情報が見つかりませんでした"
            return [error]
        }
        this.attached_kyou = res.kyou_histories[0]
        return new Array<GkillError>()

    }

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetReKyouRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_rekyou(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.rekyou_histories
        return new Array<GkillError>()
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        return this.clear_attached_histories()
    }

    async clear_attached_kyou(): Promise<Array<GkillError>> {
        this.attached_kyou = null
        return new Array<GkillError>()
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        this.attached_histories = []
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        this.attached_tags = []
        this.attached_texts = []
        this.attached_notifications = []
        this.attached_timeis_kyou = []
        this.attached_histories = []
        return new Array<GkillError>()
    }

    clone(): ReKyou {
        const rekyou = new ReKyou()
        rekyou.is_deleted = this.is_deleted
        rekyou.id = this.id
        rekyou.rep_name = this.rep_name
        rekyou.related_time = this.related_time
        rekyou.data_type = this.data_type
        rekyou.create_time = this.create_time
        rekyou.create_app = this.create_app
        rekyou.create_device = this.create_device
        rekyou.create_user = this.create_user
        rekyou.update_time = this.update_time
        rekyou.update_app = this.update_app
        rekyou.update_user = this.update_user
        rekyou.update_device = this.update_device
        rekyou.target_id = this.target_id
        return rekyou
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
        this.target_id = ""
        this.attached_kyou = null
        this.attached_histories = new Array<ReKyou>()
    }

}


