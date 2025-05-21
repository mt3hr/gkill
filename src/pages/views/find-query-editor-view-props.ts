'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { GkillAPI } from "@/classes/api/gkill-api"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"

export interface FindQueryEditorViewProps {
    application_config: ApplicationConfig
    gkill_api: GkillAPI
    inited: boolean
    find_kyou_query: FindKyouQuery
}
