'use strict';

import type { ApplicationConfig } from "@/classes/datas/config/application-config";
import type { GkillPropsBase } from "./gkill-props-base";

export interface miSharedTaskViewProps extends GkillPropsBase {
    app_content_height: Number;
    app_content_width: Number;
    application_config: ApplicationConfig | null
    share_id: string
}
