// ˅
'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { ServerConfig } from "@/classes/datas/config/server-config";

// ˄

export class AllocateRepDialogEmits {
    // ˅
    
    // ˄

    reveived_messages(message: Array<GkillMessage>): void {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    received_errors(errors: Array<GkillError>): void {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    requested_reload_server_config(server_config: ServerConfig): void {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
