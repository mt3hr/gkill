'use strict'

import type { ServerConfig } from "@/classes/datas/config/server-config"
import type { GkillPropsBase } from "../views/gkill-props-base"

export interface ConfirmDeleteRepDialogProps extends GkillPropsBase {
    server_configs: Array<ServerConfig>
}
