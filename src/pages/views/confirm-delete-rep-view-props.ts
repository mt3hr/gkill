'use strict'

import type { ServerConfig } from "@/classes/datas/config/server-config"
import type { GkillPropsBase } from "./gkill-props-base"
import type { Repository } from "@/classes/datas/config/repository"

export interface ConfirmDeleteRepViewProps extends GkillPropsBase {
    server_config: ServerConfig
    repository: Repository
}
