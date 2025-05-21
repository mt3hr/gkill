import type { GkillError } from "../../classes/api/gkill-error"
import type { GkillMessage } from "../../classes/api/gkill-message"

export default interface FindQueryEditorDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_close_dialog'): void
}