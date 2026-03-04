import type { KyouViewEmits } from "./kyou-view-emits";

export default interface DnoteListTableViewEmits extends KyouViewEmits {
    (e: "finish_a_aggregate_task"): void
}