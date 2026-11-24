'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetPluginListRequest extends GkillAPIRequest {
    // target_user_id が空なら自分自身のプラグインを返す。
    // 空でない指定は管理者だけが通る（アカウント管理画面からの代行用）
    target_user_id: string

    constructor() {
        super()
        this.target_user_id = ""
    }
}
