'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

// SkillReplacePlan は zip で置き換えたときに何が起きるか。
// スキル名は zip の中の SKILL.md の frontmatter の name で決まる。
export class SkillReplacePlan {
    name: string
    is_new: boolean
    added: Array<string>
    removed: Array<string>
    changed: Array<string>
    ignored: Array<string>

    constructor() {
        this.name = ""
        this.is_new = false
        this.added = new Array<string>()
        this.removed = new Array<string>()
        this.changed = new Array<string>()
        this.ignored = new Array<string>()
    }
}

export class UploadSkillResponse extends GkillAPIResponse {
    plan: SkillReplacePlan | null
    // applied は実際に置き換えたか（dry_run なら false）
    applied: boolean

    constructor() {
        super()
        this.plan = null
        this.applied = false
    }
}
