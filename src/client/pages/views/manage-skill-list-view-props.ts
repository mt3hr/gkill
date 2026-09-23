'use strict'

import type { SkillInfo } from "@/classes/api/req_res/get-skill-list-response"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ManageSkillListViewProps extends GkillPropsBase {
    skills: Array<SkillInfo>
    is_loading: boolean
    is_uploading: boolean
}
