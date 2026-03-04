'use strict'

import type { GkillError } from "../api/gkill-error"

export abstract class MetaInfoBase {
    is_deleted: boolean
    id: string
    target_id: string
    related_time: Date
    create_time: Date
    create_app: string
    create_device: string
    create_user: string
    update_time: Date
    update_app: string
    update_device: string
    update_user: string

    abstract load_attached_histories(): Promise<Array<GkillError>>
    abstract load_attached_datas(): Promise<Array<GkillError>> 
    abstract clear_attached_histories(): Promise<Array<GkillError>> 
    abstract clear_attached_datas(): Promise<Array<GkillError>> 
    abstract clone(): MetaInfoBase

    constructor() {
        this.is_deleted = false
        this.id = ""
        this.target_id = ""
        this.related_time = new Date(0)
        this.create_time = new Date(0)
        this.create_app = ""
        this.create_device = ""
        this.create_user = ""
        this.update_time = new Date(0)
        this.update_app = ""
        this.update_device = ""
        this.update_user = ""
    }
}


