'use strict'

import type { SkillInfo } from "@/classes/api/req_res/get-skill-list-response"

export interface ConfirmDeleteSkillViewEmits {
    (e: 'requested_delete_skill', skill: SkillInfo): void
    (e: 'requested_close_dialog'): void
}
