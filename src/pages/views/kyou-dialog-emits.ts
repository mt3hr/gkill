'use strict';

import type { KyouViewEmits } from "./kyou-view-emits";

export interface KyouDialogEmits extends KyouViewEmits {
    (e: 'requested_close_dialog'): void
}
