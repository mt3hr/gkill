'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

// SkillInfo はスキル一覧の1行（$GKILL_HOME/skills/<user_id>/<name>/。ADR-0634）。
export class SkillInfo {
    name: string
    description: string
    // updated_time は RFC 3339 の文字列のまま持つ（表示するときだけ Date にする）
    updated_time: string
    file_count: number
    // invalid_reason は SKILL.md が無い・frontmatter が壊れている等の理由。正常なら空文字
    invalid_reason: string

    constructor() {
        this.name = ""
        this.description = ""
        this.updated_time = ""
        this.file_count = 0
        this.invalid_reason = ""
    }
}

export class GetSkillListResponse extends GkillAPIResponse {
    skills: Array<SkillInfo>

    constructor() {
        super()
        this.skills = new Array<SkillInfo>()
    }
}
