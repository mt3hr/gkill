'use strict'

import type { SkillReplacePlan } from "@/classes/api/req_res/upload-skill-response"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ConfirmUploadSkillViewProps extends GkillPropsBase {
    plan: SkillReplacePlan
}
