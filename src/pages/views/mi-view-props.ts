'use strict';

import type { InfoIdentifier } from "@/classes/datas/info-identifier";
import type { GkillPropsBase } from "./gkill-props-base";

export interface miViewProps extends GkillPropsBase {
    highlight_targets: Array<InfoIdentifier>;
    last_added_tag: string;
}
