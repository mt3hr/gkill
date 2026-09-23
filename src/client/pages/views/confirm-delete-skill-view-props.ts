'use strict'

import type { SkillInfo } from "@/classes/api/req_res/get-skill-list-response"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ConfirmDeleteSkillViewProps extends GkillPropsBase {
    skill: SkillInfo
}
