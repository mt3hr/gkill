'use strict'

import { Account } from './account'
import { Repository } from './repository'

export class ServerConfig {

    is_loaded: boolean

    enable_this_device: boolean

    device: string

    is_local_only_access: boolean

    address: string

    enable_tls: boolean

    tls_cert_file: string

    tls_key_file: string

    open_directory_command: string

    open_file_command: string

    urlog_timeout: string

    urlog_useragent: string

    upload_size_limit_month: Number

    user_data_directory: string

    repositories: Array<Repository>

    accounts: Array<Account>

    mi_notification_public_key: string

    mi_notification_private_key: string

    async clone(): Promise<ServerConfig> {
        const server_config = new ServerConfig()
        server_config.is_loaded = this.is_loaded
        server_config.enable_this_device = this.enable_this_device
        server_config.device = this.device
        server_config.is_local_only_access = this.is_local_only_access
        server_config.address = this.address
        server_config.enable_tls = this.enable_tls
        server_config.tls_cert_file = this.tls_cert_file
        server_config.tls_key_file = this.tls_key_file
        server_config.open_directory_command = this.open_directory_command
        server_config.open_file_command = this.open_file_command
        server_config.urlog_timeout = this.urlog_timeout
        server_config.urlog_useragent = this.urlog_useragent
        server_config.upload_size_limit_month = this.upload_size_limit_month
        server_config.user_data_directory = this.user_data_directory
        server_config.repositories = this.repositories
        server_config.accounts = this.accounts
        server_config.mi_notification_public_key = this.mi_notification_public_key
        server_config.mi_notification_private_key = this.mi_notification_private_key
        return server_config
    }

    constructor() {
        this.is_loaded = false
        this.enable_this_device = false
        this.device = ""
        this.is_local_only_access = true
        this.address = "8888"
        this.enable_tls = false
        this.tls_cert_file = ""
        this.tls_key_file = ""
        this.open_directory_command = ""
        this.open_file_command = ""
        this.urlog_timeout = "10s"
        this.urlog_useragent = ""
        this.upload_size_limit_month = 0
        this.user_data_directory = ""
        this.mi_notification_public_key = ""
        this.mi_notification_private_key = ""
        this.repositories = new Array<Repository>()
        this.accounts = new Array<Account>()
    }

}


