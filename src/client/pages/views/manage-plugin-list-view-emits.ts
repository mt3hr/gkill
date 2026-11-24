'use strict'

import type { PluginInfo } from "@/classes/api/req_res/get-plugin-list-response"

export interface ManagePluginListViewEmits {
    (e: 'requested_show_plugin_config', plugin: PluginInfo): void
}
