'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";

export interface SelectUploadFileTargetRepDialogEmits {
    (e: 'reveived_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_update_upload_file_target_rep', rep_name: string): void
    (e: 'requested_close_dialog'): void
}
