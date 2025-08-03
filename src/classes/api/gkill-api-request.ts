'use strict'

import { GkillAPI } from "./gkill-api"

export class GkillAPIRequest {

    abort_controller: AbortController

    session_id: string

    force_reget: boolean

    constructor() {
        try {
            this.session_id = GkillAPI.get_instance().get_session_id()
        } catch (e: any) {
            // Shareからだとdocumentがなくて例外が飛ぶ。ブランクセット、リクエスト生成後に明示的に入れてもらうようにする
            this.session_id = ""
        }
        this.abort_controller = new AbortController()
        this.force_reget = false
    }
}