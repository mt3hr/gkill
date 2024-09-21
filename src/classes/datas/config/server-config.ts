'use strict'

import { Account } from './account'
import { Repository } from './repository'

export class ServerConfig {

    device: string

    is_local_only_access: boolean

    address: string

    enable_tls: boolean

    tls_cert_file: string

    tls_key_file: string

    open_directory_command: string

    open_file_command: string

    urlog_timeout: Number

    urlog_useragent: string

    upload_size_limit_month: Number

    user_data_directory: string

    repositories: Array<Repository>

    accounts: Array<Account>

    async clone(): Promise<ServerConfig> {
        throw new Error('Not implemented')
    }

    constructor() {
        this.device = ""
        this.is_local_only_access = true
        this.address = "8888"
        this.enable_tls = false
        this.tls_cert_file = ""
        this.tls_key_file = ""
        this.open_directory_command = ""
        this.open_file_command = ""
        this.urlog_timeout = 0
        this.urlog_useragent = ""
        this.upload_size_limit_month = 0
        this.user_data_directory = ""
        this.repositories = new Array<Repository>()
        this.accounts = new Array<Account>()
    }

}


