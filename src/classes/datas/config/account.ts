'use strict'

export class Account {

    user_id: string

    is_admin: boolean

    is_enable: boolean

    password_reset_token: string | null

    constructor() {
        this.user_id = ""
        this.is_admin = false
        this.is_enable = false
        this.password_reset_token = null
    }

}


