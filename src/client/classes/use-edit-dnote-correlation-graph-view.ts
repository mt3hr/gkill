import { computed } from 'vue'
import type { DnoteCorrelationGraphQuery } from '@/classes/dnote/dnote-correlation'
import type EditDnoteCorrelationGraphViewEmits from '@/pages/views/edit-dnote-correlation-graph-view-emits'
import type EditDnoteCorrelationGraphViewProps from '@/pages/views/edit-dnote-correlation-graph-view-props'

export function useEditDnoteCorrelationGraphView(options: {
    props: EditDnoteCorrelationGraphViewProps,
    emits: EditDnoteCorrelationGraphViewEmits,
}) {
    const { props, emits } = options

    // ── Computed ──
    // 複製はフォーム側(useDnoteCorrelationGraphEditorView)が行う。
    // ここで複製すると、propsが差し替わるたびに別インスタンスになって
    // フォームの読み直しが二重に走る
    const initial_query = computed(() => props.dnote_correlation_graph_query)

    // ── Business logic ──
    function save(dnote_correlation_graph_query: DnoteCorrelationGraphQuery): void {
        emits('requested_update_dnote_correlation_graph', dnote_correlation_graph_query)
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // Computed
        initial_query,

        // Business logic
        save,
    }
}
