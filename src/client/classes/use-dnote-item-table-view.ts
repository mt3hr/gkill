import { nextTick, ref, type Ref } from 'vue'
import type DnoteItem from '@/classes/dnote/dnote-item'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type DnoteItemTableViewEmits from '@/pages/views/dnote-item-table-view-emits'
import type DnoteItemTableViewProps from '@/pages/views/dnote-item-table-view-props'
import type { ComponentRef } from '@/classes/component-ref'
import type { GkillError } from '@/classes/api/gkill-error'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'

export function useDnoteItemTableView(options: {
    props: DnoteItemTableViewProps,
    emits: DnoteItemTableViewEmits,
    model_value: Ref<Array<Array<DnoteItem>>>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const dnote_item_list_views = ref<ComponentRef | null>(null)

    // ── Methods ──
    async function load_aggregated_value(
        abort_controller: AbortController,
        kyous: Array<Kyou>,
        query: FindKyouQuery,
        kyou_is_loaded: boolean
    ) {
        if (!dnote_item_list_views.value) return
        const wait_promises: Array<Promise<Array<GkillError>>> = []
        for (let i = 0; i < dnote_item_list_views.value.length; i++) {
            const v = dnote_item_list_views.value[i]
            if (!v) continue
            wait_promises.push(v.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded))
        }
        await Promise.all(wait_promises)
    }

    async function reset(): Promise<void> {
        if (!dnote_item_list_views.value || dnote_item_list_views.value.length === 0) return
        return nextTick(async () => {
            for (let i = 0; i < dnote_item_list_views.value!.length; i++) {
                await dnote_item_list_views.value![i].reset()
            }
        })
    }

    function handle_move_dnote_item(
        src_id: string,
        src_list_index: number,
        target_id: string | null,
        target_list_index: number,
        drop_type: "up" | "down"
    ): void {
        if (!props.editable) return

        const src_list = model_value.value[src_list_index]
        const target_list = model_value.value[target_list_index]
        if (!src_list || !target_list) return

        const src_pos = src_list.findIndex((x) => x.id === src_id)
        if (src_pos < 0) return
        const [moved] = src_list.splice(src_pos, 1)

        let insert_pos = 0
        if (target_id) {
            const target_pos = target_list.findIndex((x) => x.id === target_id)
            insert_pos = target_pos < 0 ? (drop_type === "up" ? 0 : target_list.length) : (drop_type === "up" ? target_pos : target_pos + 1)
        } else {
            insert_pos = drop_type === "up" ? 0 : target_list.length
        }

        if (src_list_index === target_list_index && src_pos < insert_pos) insert_pos -= 1
        if (insert_pos < 0) insert_pos = 0
        if (insert_pos > target_list.length) insert_pos = target_list.length
        target_list.splice(insert_pos, 0, moved)
    }

    function onCellDragover(e: DragEvent): void {
        if (!props.editable) return
        e.preventDefault()
        if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
    }

    function onCellDrop(e: DragEvent, target_list_index: number): void {
        if (!props.editable) return

        const src_id = e.dataTransfer?.getData("gkill_dnote_item_id")
        const src_list_index_str = e.dataTransfer?.getData("gkill_dnote_item_src_list_index")
        if (!src_id || src_list_index_str === undefined || src_list_index_str === null || src_list_index_str === "") return

        const src_list_index = Number(src_list_index_str)
        const el = e.currentTarget as HTMLElement | null
        if (!el) return
        const rect = el.getBoundingClientRect()
        const y = e.clientY - rect.top
        const drop_type: "up" | "down" = y <= rect.height * 0.5 ? "up" : "down"

        handle_move_dnote_item(src_id, src_list_index, null, target_list_index, drop_type)
        e.preventDefault()
        e.stopPropagation()
    }

    // ── Event relay objects ──
    // 手書きで17個並べていた頃は requested_reload_kyou / requested_reload_list /
    // requested_update_check_kyous を落としていた。
    // 自分ではフォーカスを発火しない中間層なので dialog 版（focus系込み）を使う
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        // クリックはフォーカス移動も伴う
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })

    // ── Return ──
    return {
        // Template refs
        dnote_item_list_views,

        // Methods used in template
        handle_move_dnote_item,
        onCellDragover,
        onCellDrop,

        // Exposed methods
        load_aggregated_value,
        reset,

        // Event relay objects
        crudRelayHandlers,
    }
}
