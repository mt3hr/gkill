'use strict'

import type { ServerConfig } from "@/classes/datas/config/server-config"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ServerConfigViewProps extends GkillPropsBase {
    server_configs: Array<ServerConfig>
}
