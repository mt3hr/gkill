'use strict';

import type { LantanaTextData } from "@/classes/lantana/lantana-text-data";

export interface LantanaTextViewEmits {
    (e: 'requested_delete_lantana_text', lantana_text: LantanaTextData): void
    (e: 'updated_lantana_text', lantana_text: LantanaTextData): void
    (e: 'request_delete_lantana_text'): void
    (e: 'update_lantana_text'): void
}
