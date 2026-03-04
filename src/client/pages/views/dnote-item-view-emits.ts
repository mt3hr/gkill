import type DnoteItem from "@/classes/dnote/dnote-item";
import type { KyouViewEmits } from "./kyou-view-emits";

export default interface DnoteItemViewEmits extends KyouViewEmits {
    (e: 'requested_delete_dnote_item', dnote_item_id: string): void
    (e: 'requested_update_dnote_item', dnote_item: DnoteItem): void
    (e: 'finish_a_aggregate_task'): void
    (e: "requested_move_dnote_item", srcId: string, srcListIndex: number, targetId: string | null, targetListIndex: number, dropType: "up" | "down"): void
}