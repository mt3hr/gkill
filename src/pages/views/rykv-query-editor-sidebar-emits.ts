'use strict';

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";

export interface rykvQueryEditorSidebarEmits {
    (e: 'updated_query', query: FindKyouQuery): void
    (e: 'request_search'): void
}
