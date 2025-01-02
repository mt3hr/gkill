'use strict'

import { GkillAPI } from "../api/gkill-api"
import { GkillError } from "../api/gkill-error"
import { GetKyousRequest } from "../api/req_res/get-kyous-request"
import { GetTagsByTargetIDRequest } from "../api/req_res/get-tags-by-target-id-request"
import { GetTextsByTargetIDRequest } from "../api/req_res/get-texts-by-target-id-request"
import type { Kyou } from "./kyou"
import type { Tag } from "./tag"
import type { Text } from "./text"

export abstract class InfoBase {
    is_deleted: boolean
    id: string
    rep_name: string
    related_time: Date
    data_type: string
    create_time: Date
    create_app: string
    create_device: string
    create_user: string
    update_time: Date
    update_app: string
    update_user: string
    update_device: string
    attached_tags: Array<Tag>
    attached_texts: Array<Text>
    attached_timeis_kyou: Array<Kyou>
    is_checked: boolean

    async load_all(): Promise<Array<GkillError>> {
        return this.load_attached_datas()
    }

    async load_attached_tags(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        const req = new GetTagsByTargetIDRequest()
        req.session_id = GkillAPI.get_gkill_api().get_session_id()
        req.target_id = this.id
        const res = await GkillAPI.get_gkill_api().get_tags_by_target_id(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_tags = res.tags
        return errors
    }

    async load_attached_texts(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        const req = new GetTextsByTargetIDRequest()
        req.session_id = GkillAPI.get_gkill_api().get_session_id()
        req.target_id = this.id
        const res = await GkillAPI.get_gkill_api().get_texts_by_target_id(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_texts = res.texts
        return errors
    }

    async load_attached_timeis(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()

        const application_config = GkillAPI.get_gkill_api().get_saved_application_config()
        if (!application_config) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "設定読込が完了していません"
            return errors
        }

        let reps = new Array<string>()
        let tags = new Array<string>()
        for (let i = 0; i < application_config.rep_struct.length; i++) {
            reps.push(application_config.rep_struct[i].rep_name)
        }
        for (let i = 0; i < application_config.tag_struct.length; i++) {
            tags.push(application_config.tag_struct[i].tag_name)
        }

        const req = new GetKyousRequest()
        req.session_id = GkillAPI.get_gkill_api().get_session_id()
        req.query.use_plaing = true
        req.query.plaing_time = this.related_time
        req.query.reps = reps
        req.query.tags = tags

        const res = await GkillAPI.get_gkill_api().get_kyous(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        for (let i = 0; i < res.kyous.length; i++) {
            await res.kyous[i].load_typed_timeis()
        }
        this.attached_timeis_kyou = res.kyous
        return errors
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        const awaitPromises = new Array<Promise<any>>()
        awaitPromises.push(this.load_attached_tags())
        awaitPromises.push(this.load_attached_texts())
        awaitPromises.push(this.load_attached_timeis())
        return Promise.all(awaitPromises).then((errors_list) => {
            const errors = new Array<GkillError>()
            errors_list.forEach(e => {
                errors.push(...e)
            })
            return errors
        })
    }

    async clear_attached_tags(): Promise<Array<GkillError>> {
        this.attached_tags = []
        return new Array<GkillError>()
    }

    async clear_attached_texts(): Promise<Array<GkillError>> {
        this.attached_texts = []
        return new Array<GkillError>()
    }

    async clear_attached_timeis(): Promise<Array<GkillError>> {
        this.attached_timeis_kyou = []
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        this.attached_tags = []
        this.attached_texts = []
        this.attached_timeis_kyou = []
        return new Array<GkillError>()
    }

    abstract clone(): InfoBase

    constructor() {
        this.is_deleted = false
        this.id = ""
        this.rep_name = ""
        this.related_time = new Date(0)
        this.data_type = ""
        this.create_time = new Date(0)
        this.create_app = ""
        this.create_device = ""
        this.create_user = ""
        this.update_time = new Date(0)
        this.update_app = ""
        this.update_user = ""
        this.update_device = ""
        this.attached_tags = new Array<Tag>()
        this.attached_texts = new Array<Text>()
        this.attached_timeis_kyou = new Array<Kyou>()
        this.is_checked = false
    }
}


