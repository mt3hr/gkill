'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

export class GetSkillRequest extends GkillAPIRequest {
    name: string
    // path を空にすると SKILL.md の全文とファイル一覧、指定するとそのファイルの中身が返る
    path: string
    // max_bytes は AI（MCP）が渡す量を抑えるためのもの。画面は 0（上限なし）で呼ぶ
    max_bytes: number

    constructor() {
        super()
        this.name = ""
        this.path = ""
        this.max_bytes = 0
    }
}
