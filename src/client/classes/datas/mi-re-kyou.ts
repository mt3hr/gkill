'use strict'

import { GkillAPI } from '../api/gkill-api'
import { GkillError } from '../api/gkill-error'
import { i18n } from '@/i18n'
import { GkillErrorCodes } from '../api/message/gkill_error'
import { GetKyouRequest } from '../api/req_res/get-kyou-request'
import { GetMiReKyouRequest } from '../api/req_res/get-mi-re-kyou-request'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'
import { Kyou } from './kyou'

/**
 * MiReKyou は既存Kyouをタスク化した情報。
 * ReKyou由来のtarget_idと、Mi由来のスケジュール項目を併せ持つ。
 * タイトルは持たず、表示時はtarget_idの指すKyouをそのまま描画する。
 */
export class MiReKyou extends InfoBase {

    target_id: string

    is_checked: boolean

    board_name: string

    limit_time: Date | null

    estimate_start_time: Date | null

    estimate_end_time: Date | null

    attached_kyou: Kyou | null

    attached_histories: Array<MiReKyou>

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
            error.error_code = GkillErrorCodes.not_found_mi_rekyou_target
            error.error_message = i18n.global.t('NOT_FOUND_MI_REKYOU_TARGET_ERROR_MESSAGE')
            return [error]
        }
        this.attached_kyou = res.kyou_histories[0]
        return new Array<GkillError>()
    }

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetMiReKyouRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_mirekyou(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.mirekyou_histories
        return new Array<GkillError>()
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        // MiReKyouの付随データはtarget_id先(集約側)が読むため、ここでは自前ロードせず履歴を消すだけでよい
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

    clone(): MiReKyou {
        const mirekyou = new MiReKyou()
        mirekyou.is_deleted = this.is_deleted
        mirekyou.id = this.id
        mirekyou.rep_name = this.rep_name
        mirekyou.related_time = this.related_time
        mirekyou.data_type = this.data_type
        mirekyou.create_time = this.create_time
        mirekyou.create_app = this.create_app
        mirekyou.create_device = this.create_device
        mirekyou.create_user = this.create_user
        mirekyou.update_time = this.update_time
        mirekyou.update_app = this.update_app
        mirekyou.update_user = this.update_user
        mirekyou.update_device = this.update_device
        mirekyou.target_id = this.target_id
        mirekyou.is_checked = this.is_checked
        mirekyou.board_name = this.board_name
        mirekyou.limit_time = this.limit_time
        mirekyou.estimate_start_time = this.estimate_start_time
        mirekyou.estimate_end_time = this.estimate_end_time
        return mirekyou
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
        this.is_checked = false
        this.board_name = ""
        this.limit_time = null
        this.estimate_start_time = null
        this.estimate_end_time = null
        this.attached_kyou = null
        this.attached_histories = new Array<MiReKyou>()
    }

}
