'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

export class GetGkillInfoResponse extends GkillAPIResponse {

    user_id: string

    device: string

    user_is_admin: boolean

    cache_clear_count_limit: number

    global_ip: string

    private_ip: string

	version: string

	build_time: Date

	commit_hash: string

    constructor() {
        super()
        this.user_id = ""
        this.device = ""
        this.user_is_admin = false
        this.cache_clear_count_limit = 1001
        this.global_ip = ""
        this.private_ip = ""
        this.version = ""
        this.build_time = new Date(0)
        this.commit_hash = ""
    }

}


