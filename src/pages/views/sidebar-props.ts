'use strict';

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { GkillPropsBase } from "./gkill-props-base";

export interface SidebarProps extends GkillPropsBase {
    query: FindKyouQuery;
}
