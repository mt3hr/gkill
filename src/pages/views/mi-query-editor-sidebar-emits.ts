'use strict';

import type { FindMiQuery } from "@/classes/api/find_query/find-mi-query";

export interface miQueryEditorSidebarEmits {
    (e: 'updated_query', query: FindMiQuery): void
}
