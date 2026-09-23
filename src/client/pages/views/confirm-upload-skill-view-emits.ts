'use strict'

import type { SkillReplacePlan } from "@/classes/api/req_res/upload-skill-response"

export interface ConfirmUploadSkillViewEmits {
    (e: 'requested_apply_upload_skill', plan: SkillReplacePlan): void
    (e: 'requested_close_dialog'): void
}
