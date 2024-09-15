'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { Account } from "@/classes/datas/config/account";
import type { ServerConfig } from "@/classes/datas/config/server-config";

export interface CreateAccountDialogEmits {
    (e: 'reveived_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_reload_server_config', server_config: ServerConfig): void
    (e: 'added_account', account: Account): void
}
