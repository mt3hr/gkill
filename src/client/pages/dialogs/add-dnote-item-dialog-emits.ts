import type { GkillError } from "../../classes/api/gkill-error"
import type { GkillMessage } from "../../classes/api/gkill-message"
import type DnoteItem from "../../classes/dnote/dnote-item"

export default interface AddDnoteItemDialogEmits {
    (e: 'requested_add_dnote_item', dnote_item: DnoteItem): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}