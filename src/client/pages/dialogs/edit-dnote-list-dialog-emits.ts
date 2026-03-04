import type { GkillError } from "../../classes/api/gkill-error"
import type { GkillMessage } from "../../classes/api/gkill-message"
import type DnoteListQuery from "../views/dnote-list-query"

export default interface EditDnoteListDialogEmits {
    (e: 'requested_update_dnote_list_query', dnote_list_query: DnoteListQuery): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}