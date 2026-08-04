'use strict'

import type { MiCheckState } from "@/classes/api/find_query/mi-check-state"

export interface miExtractCheckStateQueryEmits {
    (e: 'request_update_extract_check_state', check_state: MiCheckState): void
    (e: 'request_clear_check_state'): void
    (e: 'inited'): void
}
