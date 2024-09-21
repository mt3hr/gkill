'use strict';

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { GkillPropsBase } from "./gkill-props-base";

export interface DnoteProps extends GkillPropsBase {
    query: FindKyouQuery;
    app_content_height: Number;
    app_content_width: Number;
    last_added_tag: string;
}
