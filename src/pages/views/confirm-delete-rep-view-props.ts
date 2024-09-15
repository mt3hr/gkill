'use strict';

import type { ServerConfig } from "@/classes/datas/config/server-config";
import type { GkillPropsBase } from "./gkill-props-base";

export interface ConfirmDeleteRepViewProps extends GkillPropsBase {
    server_config: ServerConfig;
    rep_id: string;
}