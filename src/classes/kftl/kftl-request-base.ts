'use strict'

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
        throw new Error('Not implemented')
    }

    async get_create_time(): Promise<Date> {
        throw new Error('Not implemented')
    }

    async set_create_time(create_time: Date): Promise<void> {
        throw new Error('Not implemented')
    }

    async get_create_app(): Promise<string> {
        throw new Error('Not implemented')
    }

    async set_create_app(create_app: string): Promise<void> {
        throw new Error('Not implemented')
    }

    async get_create_device(): Promise<string> {
        throw new Error('Not implemented')
    }

    async set_create_device(create_device: string): Promise<void> {
        throw new Error('Not implemented')
    }

    async get_create_user(): Promise<string> {
        throw new Error('Not implemented')
    }

    async set_create_user(create_user: string): Promise<void> {
        throw new Error('Not implemented')
    }

    async get_update_time(): Promise<Date> {
        throw new Error('Not implemented')
    }

    async set_update_time(create_time: Date): Promise<void> {
        throw new Error('Not implemented')
    }

    async get_update_app(): Promise<string> {
        throw new Error('Not implemented')
    }

    async set_update_app(create_app: string): Promise<void> {
        throw new Error('Not implemented')
    }

    async get_update_device(): Promise<string> {
        throw new Error('Not implemented')
    }

    async set_update_device(create_device: string): Promise<void> {
        throw new Error('Not implemented')
    }

    async get_update_user(): Promise<string> {
        throw new Error('Not implemented')
    }

    async set_update_user(create_user: string): Promise<void> {
        throw new Error('Not implemented')
    }

}


