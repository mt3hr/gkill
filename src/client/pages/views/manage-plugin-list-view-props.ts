'use strict'

import type { PluginInfo } from "@/classes/api/req_res/get-plugin-list-response"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ManagePluginListViewProps extends GkillPropsBase {
    target_user_id: string
    plugins: Array<PluginInfo>
    is_loading: boolean
}
