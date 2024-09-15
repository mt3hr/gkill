'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { KFTLTemplateElement } from "@/classes/datas/kftl-template-element";

export interface KFTLTemplateDialogEmits {
    (e: 'reveived_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'clicked_template_element_leaf', template_leaf: KFTLTemplateElement): void
    (e: 'closed_dialog'): void
}
