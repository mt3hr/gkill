import type { KyouViewEmits } from "./kyou-view-emits";

export default interface DnoteItemListViewEmits extends KyouViewEmits {
    (e: "finish_a_aggregate_task"): void
    (e: 'requested_move_dnote_item', srcId: string, srcListIndex: number, targetId: string | null, targetListIndex: number, dropType: 'up' | 'down'): void
}