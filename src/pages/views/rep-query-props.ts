'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { GkillAPI } from "@/classes/api/gkill-api"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"
import type { SidebarProps } from "./sidebar-props"

export interface RepQueryProps extends SidebarProps {
    query: FindKyouQuery
    application_config: ApplicationConfig
    gkill_api: GkillAPI
}
