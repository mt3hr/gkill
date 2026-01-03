'use strict'

import { GkillAPI } from "../api/gkill-api"
import { GkillError } from "../api/gkill-error"
import { GetApplicationConfigRequest } from "../api/req_res/get-application-config-request"
import { GetKyousRequest } from "../api/req_res/get-kyous-request"
import { GetNotificationsByTargetIDRequest } from "../api/req_res/get-notifications-by-target-id-request"
import { GetTagsByTargetIDRequest } from "../api/req_res/get-tags-by-target-id-request"
import { GetTextsByTargetIDRequest } from "../api/req_res/get-texts-by-target-id-request"
import type { RepStructElementData } from "./config/rep-struct-element-data"
import type { TagStructElementData } from "./config/tag-struct-element-data"
import type { Kyou } from "./kyou"
import type { Notification } from "./notification"
import type { Tag } from "./tag"
import type { Text } from "./text"

export abstract class InfoBase {
    abort_controller: AbortController
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
    attached_notifications: Array<Notification>
    attached_timeis_kyou: Array<Kyou>
    is_checked_kyou: boolean

    async load_all(): Promise<Array<GkillError>> {
        return this.load_attached_datas()
    }

    async load_attached_tags(): Promise<Array<GkillError>> {
        const errors = new Array<GkillError>()
        const req = new GetTagsByTargetIDRequest()
        req.abort_controller = this.abort_controller

        req.target_id = this.id
        const res = await GkillAPI.get_gkill_api().get_tags_by_target_id(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_tags = res.tags
        return errors
    }

    async load_attached_texts(): Promise<Array<GkillError>> {
        const errors = new Array<GkillError>()
        const req = new GetTextsByTargetIDRequest()
        req.abort_controller = this.abort_controller

        req.target_id = this.id
        const res = await GkillAPI.get_gkill_api().get_texts_by_target_id(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_texts = res.texts
        return errors
    }

    async load_attached_notifications(): Promise<Array<GkillError>> {
        const errors = new Array<GkillError>()
        const req = new GetNotificationsByTargetIDRequest()
        req.abort_controller = this.abort_controller

        req.target_id = this.id
        const res = await GkillAPI.get_gkill_api().get_notifications_by_target_id(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_notifications = res.notifications
        return errors
    }

    async load_attached_timeis(): Promise<Array<GkillError>> {
        const errors = new Array<GkillError>()
        const req = new GetKyousRequest()
        req.abort_controller = this.abort_controller

        req.query.use_tags = false
        req.query.use_plaing = true
        req.query.plaing_time = this.related_time

        const application_config = (await GkillAPI.get_gkill_api().get_application_config(new GetApplicationConfigRequest())).application_config
        if (!application_config.for_share_kyou) {
            let rep_name_walk = (_rep: RepStructElementData): Array<string> => []
            rep_name_walk = (rep: RepStructElementData): Array<string> => {
                const rep_names = new Array<string>()
                const rep_children = rep.children
                if (rep_children) {
                    rep_children.forEach(child_rep => {
                        rep_names.push(child_rep.rep_name)
                        if (child_rep) {
                            rep_names.push(...rep_name_walk(child_rep))
                        }
                    })
                }
                return rep_names
            }
            req.query.reps = rep_name_walk(application_config.rep_struct)
            let tag_name_walk = (_tag: TagStructElementData): Array<string> => []
            tag_name_walk = (tag: TagStructElementData): Array<string> => {
                const tag_names = new Array<string>()
                const tag_children = tag.children
                if (tag_children) {
                    tag_children.forEach(child_tag => {
                        if (child_tag.check_when_inited) {
                            tag_names.push(child_tag.tag_name)
                        }
                        if (child_tag) {
                            tag_names.push(...tag_name_walk(child_tag))
                        }
                    })
                }
                return tag_names
            }
            req.query.tags = tag_name_walk(application_config.tag_struct)
        }

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
        awaitPromises.push(this.load_attached_notifications())
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

    async clear_attached_notifications(): Promise<Array<GkillError>> {
        this.attached_notifications = []
        return new Array<GkillError>()
    }

    async clear_attached_timeis(): Promise<Array<GkillError>> {
        this.attached_timeis_kyou = []
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        this.attached_tags = []
        this.attached_texts = []
        this.attached_notifications = []
        this.attached_timeis_kyou = []
        return new Array<GkillError>()
    }

    abstract clone(): InfoBase

    constructor() {
        this.abort_controller = new AbortController()
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
        this.attached_notifications = new Array<Notification>()
        this.attached_timeis_kyou = new Array<Kyou>()
        this.is_checked_kyou = false
    }
}