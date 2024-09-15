'use strict';

import type { LantanaTextType } from "@/classes/lantana/lantana-text-type";

export interface LantanaTextTypeSelectBoxEmits {
    (e: 'updated_text_type',text_type: LantanaTextType): void 
}
