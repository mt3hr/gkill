import { nextTick, ref, type Ref } from "vue"
import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { Kyou } from "@/classes/datas/kyou"
import type { ComponentRef } from "@/classes/component-ref"
import type { DnoteCorrelationGraphQuery } from "@/classes/dnote/dnote-correlation"
import type DnoteCorrelationGraphTableViewEmits from "@/pages/views/dnote-correlation-graph-table-view-emits"
import type DnoteCorrelationGraphTableViewProps from "@/pages/views/dnote-correlation-graph-table-view-props"

export function useDnoteCorrelationGraphTableView(options: {
    props: DnoteCorrelationGraphTableViewProps,
    emits: DnoteCorrelationGraphTableViewEmits,
    model_value: Ref<Array<DnoteCorrelationGraphQuery>>,
}) {
    const { props, model_value } = options

    // ── Template refs ──
    const dnote_correlation_graph_views = ref<ComponentRef | null>(null)

    // ── Methods ──
    async function load_correlation(
        abort_controller: AbortController,
        kyous: Array<Kyou>,
        query: FindKyouQuery,
        kyou_is_loaded: boolean
    ): Promise<void> {
        if (!dnote_correlation_graph_views.value) return
        const waits: Array<Promise<void>> = []
        for (let i = 0; i < dnote_correlation_graph_views.value.length; i++) {
            const v = dnote_correlation_graph_views.value[i]
            if (!v) continue
            waits.push(v.load_correlation(abort_controller, kyous, query, kyou_is_loaded))
        }
        await Promise.all(waits)
    }

    async function reset(): Promise<void> {
        if (!dnote_correlation_graph_views.value || dnote_correlation_graph_views.value.length === 0) return
        return nextTick(async () => {
            for (let i = 0; i < dnote_correlation_graph_views.value!.length; i++) await dnote_correlation_graph_views.value![i].reset()
        })
    }

    function delete_dnote_correlation_graph(id: string): void {
        const idx = model_value.value.findIndex((x) => x.id === id)
        if (idx < 0) return
        model_value.value.splice(idx, 1)
    }

    function update_dnote_correlation_graph(q: DnoteCorrelationGraphQuery): void {
        const idx = model_value.value.findIndex((x) => x.id === q.id)
        if (idx < 0) return
        model_value.value.splice(idx, 1, q)
    }

    function handle_move_dnote_correlation_graph(src_id: string, target_id: string, drop_type: "left" | "right"): void {
        if (!props.editable) return
        const src_index = model_value.value.findIndex((x) => x.id === src_id)
        if (src_index < 0) return
        const [moved] = model_value.value.splice(src_index, 1)

        const target_index = model_value.value.findIndex((x) => x.id === target_id)
        if (target_index < 0) { model_value.value.push(moved); return }

        // 先に取り除いてあるので、移動元より後ろへ挿す場合は添字が1つ前へずれる
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
        const src_id = e.dataTransfer?.getData("gkill_dnote_correlation_graph_id")
        if (!src_id) return

        const src_index = model_value.value.findIndex((x) => x.id === src_id)
        if (src_index < 0) return
        const [moved] = model_value.value.splice(src_index, 1)

        // 縦積みのため上下で先頭/末尾を判定する
        const el = e.currentTarget as HTMLElement | null
        if (!el) return
        const rect = el.getBoundingClientRect()
        const y = e.clientY - rect.top
        const insert_index = y <= rect.height * 0.5 ? 0 : model_value.value.length
        model_value.value.splice(insert_index, 0, moved)

        e.preventDefault()
        e.stopPropagation()
    }

    // ── Return ──
    return {
        // Template refs
        dnote_correlation_graph_views,

        // Methods used in template
        handle_move_dnote_correlation_graph,
        delete_dnote_correlation_graph,
        update_dnote_correlation_graph,
        onTableDragover,
        onTableDrop,

        // Exposed methods
        load_correlation,
        reset,
    }
}
