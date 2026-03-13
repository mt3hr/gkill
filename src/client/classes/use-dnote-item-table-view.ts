import { nextTick, ref, type Ref } from 'vue'
import type DnoteItem from '@/classes/dnote/dnote-item'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type DnoteItemTableViewEmits from '@/pages/views/dnote-item-table-view-emits'
import type DnoteItemTableViewProps from '@/pages/views/dnote-item-table-view-props'

export function useDnoteItemTableView(options: {
    props: DnoteItemTableViewProps,
    emits: DnoteItemTableViewEmits,
    model_value: Ref<Array<Array<DnoteItem>>>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const dnote_item_list_views = ref<any>(null)

    // ── Methods ──
    async function load_aggregated_value(
        abort_controller: AbortController,
        kyous: Array<Kyou>,
        query: FindKyouQuery,
        kyou_is_loaded: boolean
    ) {
        if (!dnote_item_list_views.value) return
        const waitPromises: Array<Promise<any>> = []
        for (let i = 0; i < dnote_item_list_views.value.length; i++) {
            const v = dnote_item_list_views.value[i]
            if (!v) continue
            waitPromises.push(v.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded))
        }
        await Promise.all(waitPromises)
    }

    async function reset(): Promise<void> {
        if (!dnote_item_list_views.value || dnote_item_list_views.value.length === 0) return
        return nextTick(async () => {
            for (let i = 0; i < dnote_item_list_views.value.length; i++) {
                await dnote_item_list_views.value[i].reset()
            }
        })
    }

    function handle_move_dnote_item(
        srcId: string,
        srcListIndex: number,
        targetId: string | null,
        targetListIndex: number,
        dropType: "up" | "down"
    ): void {
        if (!props.editable) return

        const srcList = model_value.value[srcListIndex]
        const targetList = model_value.value[targetListIndex]
        if (!srcList || !targetList) return

        const srcPos = srcList.findIndex((x) => x.id === srcId)
        if (srcPos < 0) return
        const [moved] = srcList.splice(srcPos, 1)

        let insertPos = 0
        if (targetId) {
            const targetPos = targetList.findIndex((x) => x.id === targetId)
            insertPos = targetPos < 0 ? (dropType === "up" ? 0 : targetList.length) : (dropType === "up" ? targetPos : targetPos + 1)
        } else {
            insertPos = dropType === "up" ? 0 : targetList.length
        }

        if (srcListIndex === targetListIndex && srcPos < insertPos) insertPos -= 1
        if (insertPos < 0) insertPos = 0
        if (insertPos > targetList.length) insertPos = targetList.length
        targetList.splice(insertPos, 0, moved)
    }

    function on_cell_dragover(e: DragEvent): void {
        if (!props.editable) return
        e.preventDefault()
        if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
    }

    function on_cell_drop(e: DragEvent, targetListIndex: number): void {
        if (!props.editable) return

        const srcId = e.dataTransfer?.getData("gkill_dnote_item_id")
        const srcListIndexStr = e.dataTransfer?.getData("gkill_dnote_item_src_list_index")
        if (!srcId || srcListIndexStr === undefined || srcListIndexStr === null || srcListIndexStr === "") return

        const srcListIndex = Number(srcListIndexStr)
        const el = e.currentTarget as HTMLElement | null
        if (!el) return
        const rect = el.getBoundingClientRect()
        const y = e.clientY - rect.top
        const dropType: "up" | "down" = y <= rect.height * 0.5 ? "up" : "down"

        handle_move_dnote_item(srcId, srcListIndex, null, targetListIndex, dropType)
        e.preventDefault()
        e.stopPropagation()
    }

    // ── Return ──
    return {
        // Template refs
        dnote_item_list_views,

        // Methods used in template
        handle_move_dnote_item,
        on_cell_dragover,
        on_cell_drop,

        // Exposed methods
        load_aggregated_value,
        reset,
    }
}
