'use strict'

import type { GkillAPI } from "@/classes/api/gkill-api"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"
import type { SidebarProps } from "./sidebar-props"

export interface RepQueryProps extends SidebarProps {
    application_config: ApplicationConfig
    gkill_api: GkillAPI
}
