'use strict'

export class Account {

    user_id: string

    is_admin: boolean

    is_enable: boolean

    password_reset_token: string | null

    // password_reset_token_expiration はリセットトークンの有効期限。
    // get_server_configs のレスポンスは素のJSONをそのままキャストしているので
    // Date ではなくサーバが返すRFC3339文字列のまま入る
    password_reset_token_expiration: string | null

    constructor() {
        this.user_id = ""
        this.is_admin = false
        this.is_enable = false
        this.password_reset_token = null
        this.password_reset_token_expiration = null
    }

}


