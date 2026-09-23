'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

// SkillFileInfo はスキル内の1ファイル。テキストかどうかは拡張子ではなく中身でサーバが判定する。
export class SkillFileInfo {
    path: string
    size: number
    is_text: boolean
    revision: string
    updated_time: string

    constructor() {
        this.path = ""
        this.size = 0
        this.is_text = false
        this.revision = ""
        this.updated_time = ""
    }
}

// SkillDetail は1つのスキルの中身（SKILL.md の全文とファイル一覧）。
export class SkillDetail {
    name: string
    description: string
    invalid_reason: string
    updated_time: string
    // content は SKILL.md の全文（frontmatter を含む）
    content: string
    revision: string
    files: Array<SkillFileInfo>

    constructor() {
        this.name = ""
        this.description = ""
        this.invalid_reason = ""
        this.updated_time = ""
        this.content = ""
        this.revision = ""
        this.files = new Array<SkillFileInfo>()
    }
}

// SkillFileContent は1ファイルの中身。テキストは content、バイナリは content_base64。
export class SkillFileContent extends SkillFileInfo {
    content: string
    content_base64: string
    content_omitted: boolean

    constructor() {
        super()
        this.content = ""
        this.content_base64 = ""
        this.content_omitted = false
    }
}

export class GetSkillResponse extends GkillAPIResponse {
    // skill は path を空にしたときに入る
    skill: SkillDetail | null
    // file は path を指定したときに入る
    file: SkillFileContent | null

    constructor() {
        super()
        this.skill = null
        this.file = null
    }
}
