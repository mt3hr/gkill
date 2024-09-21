'use strict'

import type { MiCheckState } from "@/classes/api/find_query/mi-check-state"

export interface miExtructCheckStateQueryEmits {
    (e: 'request_update_extruct_check_state', check_state: MiCheckState): void
}
