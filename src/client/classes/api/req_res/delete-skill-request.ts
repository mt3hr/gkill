'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class DeleteSkillRequest extends GkillAPIRequest {
    name: string
    // path を空にするとスキルを丸ごと消す（画面はこちらだけを使う）
    path: string
    revision: string

    constructor() {
        super()
        this.name = ""
        this.path = ""
        this.revision = ""
    }
}
