import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface DnoteCorrelationGraphContextMenuEmits {
    (e: 'requested_edit_dnote_correlation_graph', dnote_correlation_graph_query_id: string): void
    (e: 'requested_delete_dnote_correlation_graph', dnote_correlation_graph_query_id: string): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}
