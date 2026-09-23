'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

// DownloadSkillResponse はスキルの zip を base64 で返す（gkill_fetch は JSON 以外の応答を受け付けないため）。
// zip の中身はスキル名のフォルダ1段で包まれていて、そのままアップロードし直せる。
export class DownloadSkillResponse extends GkillAPIResponse {
    file_name: string
    zip_base64: string

    constructor() {
        super()
        this.file_name = ""
        this.zip_base64 = ""
    }
}
