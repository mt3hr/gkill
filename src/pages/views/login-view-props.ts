'use strict'

import type { ApplicationConfig } from "@/classes/datas/config/application-config"
import type { GkillPropsBase } from "./gkill-props-base"
import type { GkillAPI } from "@/classes/api/gkill-api"

export interface LoginViewProps {
    app_content_height: Number
    app_content_width: Number
    gkill_api: GkillAPI
}
