import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type DnoteTrendGraphQuery from "./dnote-trend-graph-query"

export default interface DnoteTrendGraphViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_delete_dnote_trend_graph', dnote_trend_graph_query_id: string): void
    (e: 'requested_update_dnote_trend_graph', dnote_trend_graph_query: DnoteTrendGraphQuery): void
    (e: "finish_a_aggregate_task"): void
    (e: 'requested_move_dnote_trend_graph', srcId: string, targetId: string, dropType: 'left' | 'right'): void
}
