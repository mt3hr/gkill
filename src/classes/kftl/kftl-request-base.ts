'use strict'

import { GkillAPI } from "../api/gkill-api"
import { GetGkillInfoRequest } from "../api/req_res/get-gkill-info-request"

export class KFTLRequestBase {

    private is_deleted: boolean

    private create_time: Date

    private create_app: string

    private create_device: string

    private create_user: string

    private update_time: Date

    private update_app: string

    private update_device: string

    private update_user: string

    constructor() {
        this.is_deleted = false
        this.create_app = "gkill_kftl"
        this.update_app = "gkill_kftl"
        this.create_device = ""
        this.create_user = ""
        this.update_device = ""
        this.update_user = ""
        this.create_time = new Date(0)
        this.update_time = new Date(0)
    }

    async apply_default_value_kftl_request_base(): Promise<void> {
        const now = new Date(Date.now())
        const req = new GetGkillInfoRequest()

        const res = await GkillAPI.get_gkill_api().get_gkill_info(req)

        this.is_deleted = false
        this.create_app = "gkill_kftl"
        this.update_app = "gkill_kftl"
        this.create_device = res.device
        this.create_user = res.user_id
        this.update_device = ""
        this.update_user = ""
        this.create_time = now
        this.update_time = now
    }

    get_create_time(): Date {
        return this.create_time
    }

    set_create_time(create_time: Date): void {
        this.create_time = create_time
    }

    get_create_app(): string {
        return this.create_app
    }

    set_create_app(create_app: string): void {
        this.create_app = create_app
    }

    get_create_device(): string {
        return this.create_device
    }

    set_create_device(create_device: string): void {
        this.create_device = create_device
    }

    get_create_user(): string {
        return this.create_user
    }

    set_create_user(create_user: string): void {
        this.create_user = create_user
    }

    get_update_time(): Date {
        return this.update_time
    }

    set_update_time(update_time: Date): void {
        this.update_time = update_time
    }

    get_update_app(): string {
        return this.update_app
    }

    set_update_app(update_app: string): void {
        this.update_app = update_app
    }

    get_update_device(): string {
        return this.update_device
    }

    set_update_device(update_device: string): void {
        this.update_device = update_device
    }

    get_update_user(): string {
        return this.update_user
    }

    set_update_user(update_user: string): void {
        this.update_user = update_user
    }

}


