'use strict'

import type { GkillAPI } from "@/classes/api/gkill-api"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"

export interface GkillPropsBase {
    application_config: ApplicationConfig
    gkill_api: GkillAPI
}
