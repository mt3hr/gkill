import { nextTick, ref } from 'vue'
import type DnoteListQuery from '@/pages/views/dnote-list-query'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type DnoteListTableViewEmits from '@/pages/views/dnote-list-table-view-emits'
import type DnoteListTableViewProps from '@/pages/views/dnote-list-table-view-props'
import type { Ref } from 'vue'
import type { ComponentRef } from '@/classes/component-ref'
import type { GkillError } from '@/classes/api/gkill-error'

export function useDnoteListTableView(options: {
    props: DnoteListTableViewProps,
    emits: DnoteListTableViewEmits,
    model_value: Ref<Array<DnoteListQuery>>,
}) {
    const { props, emits: _emits, model_value } = options

    // ── Template refs ──
    const dnote_list_views = ref<ComponentRef | null>(null)

    // ── Methods ──
    async function load_aggregate_grouping_list(
        abort_controller: AbortController,
        kyous: Array<Kyou>,
        query: FindKyouQuery,
        kyou_is_loaded: boolean
    ): Promise<void> {
        if (!dnote_list_views.value) return
        const waits: Array<Promise<Array<GkillError>>> = []
        for (let i = 0; i < dnote_list_views.value.length; i++) {
            const v = dnote_list_views.value[i]
            if (!v) continue
            waits.push(v.load_aggregate_grouping_list(abort_controller, kyous, query, kyou_is_loaded))
        }
        await Promise.all(waits)
    }

    async function reset(): Promise<void> {
        if (!dnote_list_views.value || dnote_list_views.value.length === 0) return
        return nextTick(async () => {
            for (let i = 0; i < dnote_list_views.value!.length; i++) await dnote_list_views.value![i].reset()
        })
    }

    function delete_dnote_list_query(id: string): void {
        const idx = model_value.value.findIndex((x) => x.id === id)
        if (idx < 0) return
        model_value.value.splice(idx, 1)
    }

    function update_dnote_list_query(q: DnoteListQuery): void {
        const idx = model_value.value.findIndex((x) => x.id === q.id)
        if (idx < 0) return
        model_value.value.splice(idx, 1, q)
    }

    function handle_move_dnote_list_query(src_id: string, target_id: string, drop_type: "left" | "right"): void {
        if (!props.editable) return
        const src_index = model_value.value.findIndex((x) => x.id === src_id)
        if (src_index < 0) return
        const [moved] = model_value.value.splice(src_index, 1)

        const target_index = model_value.value.findIndex((x) => x.id === target_id)
        if (target_index < 0) { model_value.value.push(moved); return }

        let insert_index = drop_type === "left" ? target_index : target_index + 1
        if (src_index < insert_index) insert_index -= 1
        if (insert_index < 0) insert_index = 0
        if (insert_index > model_value.value.length) insert_index = model_value.value.length
        model_value.value.splice(insert_index, 0, moved)
    }

    function onTableDragover(e: DragEvent): void {
        if (!props.editable) return
        e.preventDefault()
        if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
    }

    function onTableDrop(e: DragEvent): void {
        if (!props.editable) return
        const src_id = e.dataTransfer?.getData("gkill_dnote_list_id")
        if (!src_id) return

        const src_index = model_value.value.findIndex((x) => x.id === src_id)
        if (src_index < 0) return
        const [moved] = model_value.value.splice(src_index, 1)

        const el = e.currentTarget as HTMLElement | null
        if (!el) return
        const rect = el.getBoundingClientRect()
        const x = e.clientX - rect.left
        const insert_index = x <= rect.width * 0.5 ? 0 : model_value.value.length
        model_value.value.splice(insert_index, 0, moved)

        e.preventDefault()
        e.stopPropagation()
    }

    // ── Return ──
    return {
        // Template refs
        dnote_list_views,

        // Methods used in template
        handle_move_dnote_list_query,
        delete_dnote_list_query,
        update_dnote_list_query,
        onTableDragover,
        onTableDrop,

        // Exposed methods
        load_aggregate_grouping_list,
        reset,
    }
}
