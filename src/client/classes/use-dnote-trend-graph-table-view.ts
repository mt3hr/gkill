import { nextTick, ref } from 'vue'
import type DnoteTrendGraphQuery from '@/pages/views/dnote-trend-graph-query'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type DnoteTrendGraphTableViewEmits from '@/pages/views/dnote-trend-graph-table-view-emits'
import type DnoteTrendGraphTableViewProps from '@/pages/views/dnote-trend-graph-table-view-props'
import type { Ref } from 'vue'
import type { ComponentRef } from '@/classes/component-ref'

export function useDnoteTrendGraphTableView(options: {
    props: DnoteTrendGraphTableViewProps,
    emits: DnoteTrendGraphTableViewEmits,
    model_value: Ref<Array<DnoteTrendGraphQuery>>,
}) {
    const { props, emits: _emits, model_value } = options

    // ── Template refs ──
    const dnote_trend_graph_views = ref<ComponentRef | null>(null)

    // ── Methods ──
    async function load_trend_graph(
        abort_controller: AbortController,
        kyous: Array<Kyou>,
        query: FindKyouQuery,
        kyou_is_loaded: boolean
    ): Promise<void> {
        if (!dnote_trend_graph_views.value) return
        const waits: Array<Promise<void>> = []
        for (let i = 0; i < dnote_trend_graph_views.value.length; i++) {
            const v = dnote_trend_graph_views.value[i]
            if (!v) continue
            waits.push(v.load_trend_graph(abort_controller, kyous, query, kyou_is_loaded))
        }
        await Promise.all(waits)
    }

    async function reset(): Promise<void> {
        if (!dnote_trend_graph_views.value || dnote_trend_graph_views.value.length === 0) return
        return nextTick(async () => {
            for (let i = 0; i < dnote_trend_graph_views.value!.length; i++) await dnote_trend_graph_views.value![i].reset()
        })
    }

    function delete_dnote_trend_graph(id: string): void {
        const idx = model_value.value.findIndex((x) => x.id === id)
        if (idx < 0) return
        model_value.value.splice(idx, 1)
    }

    function update_dnote_trend_graph(q: DnoteTrendGraphQuery): void {
        const idx = model_value.value.findIndex((x) => x.id === q.id)
        if (idx < 0) return
        model_value.value.splice(idx, 1, q)
    }

    function handle_move_dnote_trend_graph(srcId: string, targetId: string, dropType: "left" | "right"): void {
        if (!props.editable) return
        const srcIndex = model_value.value.findIndex((x) => x.id === srcId)
        if (srcIndex < 0) return
        const [moved] = model_value.value.splice(srcIndex, 1)

        const targetIndex = model_value.value.findIndex((x) => x.id === targetId)
        if (targetIndex < 0) { model_value.value.push(moved); return }

        let insertIndex = dropType === "left" ? targetIndex : targetIndex + 1
        if (srcIndex < insertIndex) insertIndex -= 1
        if (insertIndex < 0) insertIndex = 0
        if (insertIndex > model_value.value.length) insertIndex = model_value.value.length
        model_value.value.splice(insertIndex, 0, moved)
    }

    function onTableDragover(e: DragEvent): void {
        if (!props.editable) return
        e.preventDefault()
        if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
    }

    function onTableDrop(e: DragEvent): void {
        if (!props.editable) return
        const srcId = e.dataTransfer?.getData("gkill_dnote_trend_graph_id")
        if (!srcId) return

        const srcIndex = model_value.value.findIndex((x) => x.id === srcId)
        if (srcIndex < 0) return
        const [moved] = model_value.value.splice(srcIndex, 1)

        const el = e.currentTarget as HTMLElement | null
        if (!el) return
        // 縦積みのため上下で先頭/末尾を判定する
        const rect = el.getBoundingClientRect()
        const y = e.clientY - rect.top
        const insertIndex = y <= rect.height * 0.5 ? 0 : model_value.value.length
        model_value.value.splice(insertIndex, 0, moved)

        e.preventDefault()
        e.stopPropagation()
    }

    // ── Return ──
    return {
        // Template refs
        dnote_trend_graph_views,

        // Methods used in template
        handle_move_dnote_trend_graph,
        delete_dnote_trend_graph,
        update_dnote_trend_graph,
        onTableDragover,
        onTableDrop,

        // Exposed methods
        load_trend_graph,
        reset,
    }
}
