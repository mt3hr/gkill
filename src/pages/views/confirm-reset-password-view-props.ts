'use strict'

import type { ServerConfig } from "@/classes/datas/config/server-config"
import type { GkillPropsBase } from "./gkill-props-base"
import type { Account } from "@/classes/datas/config/account"

export interface ConfirmResetPasswordViewProps extends GkillPropsBase {
    server_config: ServerConfig
    account: Account
}
