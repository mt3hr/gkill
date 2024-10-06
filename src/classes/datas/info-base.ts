'use strict'

import type { GkillError } from "../api/gkill-error"
import type { Kyou } from "./kyou"
import type { Tag } from "./tag"
import type { Text } from "./text"
import type { TimeIs } from "./time-is"

export abstract class InfoBase {

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

    async load_attached_tags(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_attached_texts(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_attached_timeis(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_attached_tags(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_attached_texts(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_attached_timeis(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    abstract clone(): Promise<InfoBase>

    constructor() {
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


