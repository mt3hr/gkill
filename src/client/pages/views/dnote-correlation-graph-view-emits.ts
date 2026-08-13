import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { DnoteCorrelationGraphQuery } from "@/classes/dnote/dnote-correlation"

export default interface DnoteCorrelationGraphViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_delete_dnote_correlation_graph', dnote_correlation_graph_query_id: string): void
    (e: 'requested_update_dnote_correlation_graph', dnote_correlation_graph_query: DnoteCorrelationGraphQuery): void
    (e: "finish_a_aggregate_task"): void
    (e: 'requested_move_dnote_correlation_graph', srcId: string, targetId: string, dropType: 'left' | 'right'): void
}
