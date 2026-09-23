'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { SkillReplacePlan } from "@/classes/api/req_res/upload-skill-response"

export interface ConfirmUploadSkillDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_apply_upload_skill', plan: SkillReplacePlan): void
}
