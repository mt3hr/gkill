'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class UploadSkillRequest extends GkillAPIRequest {
    // zip_base64 は zip の中身。FileReader.readAsDataURL の結果（data URI）をそのまま入れてよい
    zip_base64: string
    // dry_run が true なら置き換えずに、何が起きるかだけが返る（確認の1段目）
    dry_run: boolean

    constructor() {
        super()
        this.zip_base64 = ""
        this.dry_run = false
    }
}
