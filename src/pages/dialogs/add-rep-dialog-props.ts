'use strict'

import type { ServerConfig } from "@/classes/datas/config/server-config"
import type { GkillPropsBase } from "../views/gkill-props-base"

export interface AddRepDialogProps extends GkillPropsBase {
    server_config: ServerConfig
   
}
