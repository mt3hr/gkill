'use strict'

import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { GkillAPI } from '@/classes/api/gkill-api'

export interface TutorialDialogProps {
    application_config: ApplicationConfig
    gkill_api: GkillAPI
}
