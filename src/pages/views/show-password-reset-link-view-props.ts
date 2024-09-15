'use strict';

import type { ServerConfig } from "@/classes/datas/config/server-config";
import type { GkillPropsBase } from "./gkill-props-base";

export interface ShowPasswordResetLinkViewProps extends GkillPropsBase {
    server_config: ServerConfig;
    user_id: string;
    is_show: boolean;
}
