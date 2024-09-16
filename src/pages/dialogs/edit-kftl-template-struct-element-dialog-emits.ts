'use strict';

export interface EditKFTLTemplateStructElementDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_update_kftl_template_struct_element', kftl_template_struct_element: KFTLTemplateStructElementData): void
}
