import { ref, type Ref } from 'vue'
import { i18n } from '@/i18n'
import {
    DnoteCorrelationGraphQuery,
    DnoteCorrelationMetric,
} from '@/classes/dnote/dnote-correlation'
import type AddDnoteCorrelationGraphViewEmits from '@/pages/views/add-dnote-correlation-graph-view-emits'
import type AddDnoteCorrelationGraphViewProps from '@/pages/views/add-dnote-correlation-graph-view-props'

export function useAddDnoteCorrelationGraphView(options: {
    props: AddDnoteCorrelationGraphViewProps,
    emits: AddDnoteCorrelationGraphViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    // 相関は2指標そろって初めて意味を持つので、空ではなく2件から始める
    const initial_query: Ref<DnoteCorrelationGraphQuery> = ref(new_query())

    // ── Business logic ──
    function new_metric(number: number): DnoteCorrelationMetric {
        const metric = new DnoteCorrelationMetric()
        metric.id = props.gkill_api.generate_uuid()
        metric.title = i18n.global.t("DNOTE_CORRELATION_METRIC_DEFAULT_TITLE", { number })
        return metric
    }

    function new_query(): DnoteCorrelationGraphQuery {
        const query = new DnoteCorrelationGraphQuery()
        query.id = props.gkill_api.generate_uuid()
        query.metrics = [new_metric(1), new_metric(2)]
        return query
    }

    async function reset(): Promise<void> {
        initial_query.value = new_query()
    }

    function save(dnote_correlation_graph_query: DnoteCorrelationGraphQuery): void {
        emits('requested_add_dnote_correlation_graph', dnote_correlation_graph_query)
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // State
        initial_query,

        // Business logic
        reset,
        save,
    }
}
