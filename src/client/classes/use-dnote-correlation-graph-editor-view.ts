import { ref, watch, type Ref } from "vue"
import { i18n } from "@/i18n"
import type PredicateGroupType from "@/classes/dnote/predicate-group-type"
import { predicate_struct_from_json, predicate_struct_to_json } from "@/classes/dnote/predicate-struct-json"
import type DnoteSelectItem from "@/classes/dnote/dnote-select-item"
import aggregate_target_menu_items from "@/classes/dnote/pulldown-menu/aggregate-target-menu-items"
import trend_granularity_menu_items from "@/classes/dnote/pulldown-menu/trend-granularity-menu-items"
import { build_dnote_aggregate_target_from_json, build_dnote_predicate_from_json } from "@/classes/dnote/serialize/register-dictionary"
import {
    clone_dnote_correlation_graph,
    DnoteCorrelationGraphQuery,
    DnoteCorrelationMetric,
    is_zero_based_aggregate_type,
    type DnoteCorrelationMethod,
    type DnoteTimeIsSpanPolicy,
} from "@/classes/dnote/dnote-correlation"
import type { DnoteTrendGranularity } from "@/classes/dnote/dnote-trend/dnote-trend-types"
import correlation_method_menu_items from "@/classes/dnote/pulldown-menu/correlation-method-menu-items"
import timeis_span_policy_menu_items from "@/classes/dnote/pulldown-menu/timeis-span-policy-menu-items"
import type DnoteCorrelationGraphEditorViewEmits from "@/pages/views/dnote-correlation-graph-editor-view-emits"
import type DnoteCorrelationGraphEditorViewProps from "@/pages/views/dnote-correlation-graph-editor-view-props"

// 編集中の指標。集計対象と条件は保存時に初めてクラスへ組み直すので、
// 画面上は v-select / PredicateGroup がそのまま扱える素の値で持つ
interface MetricDraft {
    id: string
    title: string
    aggregate_target: string
    root_predicate: PredicateGroupType
    missing_as_zero: boolean
    timeis_span_policy: DnoteTimeIsSpanPolicy
}

export function useDnoteCorrelationGraphEditorView(options: {
    props: DnoteCorrelationGraphEditorViewProps,
    emits: DnoteCorrelationGraphEditorViewEmits,
    initial_query: Ref<DnoteCorrelationGraphQuery>,
}) {
    const { props, emits, initial_query } = options

    // ── State refs ──
    const title = ref("")
    const granularity = ref<DnoteTrendGranularity>("day")
    const method = ref<DnoteCorrelationMethod>("pearson")
    const lag = ref(0)
    const metrics = ref<Array<MetricDraft>>([])
    const aggregate_targets: Ref<Array<DnoteSelectItem>> = ref(aggregate_target_menu_items)
    const granularities: Ref<Array<DnoteSelectItem>> = ref(trend_granularity_menu_items)
    const methods: Ref<Array<DnoteSelectItem>> = ref(correlation_method_menu_items)
    const timeis_span_policies: Ref<Array<DnoteSelectItem>> = ref(timeis_span_policy_menu_items)
    const validation_message = ref("")

    // ── Business logic ──
    function load_query(): void {
        const query = clone_dnote_correlation_graph(initial_query.value)
        title.value = query.title
        granularity.value = query.granularity
        method.value = query.method
        lag.value = query.lag
        metrics.value = query.metrics.map(metric => ({
            id: metric.id,
            title: metric.title,
            aggregate_target: String(metric.aggregate_target.to_json().type),
            root_predicate: predicate_struct_from_json(metric.predicate.predicate_struct_to_json()) as PredicateGroupType,
            missing_as_zero: metric.missing_as_zero,
            timeis_span_policy: metric.timeis_span_policy,
        }))
        validation_message.value = ""
    }

    function add_metric(): void {
        if (metrics.value.length >= 10) return
        metrics.value.push({
            id: props.gkill_api.generate_uuid(),
            title: i18n.global.t("DNOTE_CORRELATION_METRIC_DEFAULT_TITLE", { number: metrics.value.length + 1 }),
            aggregate_target: aggregate_targets.value[0].value,
            root_predicate: { logic: "AND", predicates: [] },
            missing_as_zero: false,
            timeis_span_policy: "split",
        })
    }

    // 「記録が無い期間を0とみなす」は件数・合計の集計対象でだけ意味を持つ（平均の0は観測ではない）
    function supports_missing_as_zero(draft: MetricDraft): boolean {
        return is_zero_based_aggregate_type(draft.aggregate_target)
    }

    function delete_metric(index: number): void {
        if (metrics.value.length <= 2) return
        metrics.value.splice(index, 1)
    }

    function move_metric(index: number, offset: number): void {
        const target = index + offset
        if (target < 0 || target >= metrics.value.length) return
        const [metric] = metrics.value.splice(index, 1)
        metrics.value.splice(target, 0, metric)
    }

    function save(): void {
        const trimmed_titles = metrics.value.map(metric => metric.title.trim())
        if (metrics.value.length < 2 || metrics.value.length > 10) {
            validation_message.value = i18n.global.t("DNOTE_CORRELATION_METRIC_COUNT_ERROR")
            return
        }
        if (trimmed_titles.some(metric_title => metric_title === "")) {
            validation_message.value = i18n.global.t("DNOTE_CORRELATION_METRIC_NAME_REQUIRED_ERROR")
            return
        }
        if (new Set(trimmed_titles).size !== trimmed_titles.length) {
            validation_message.value = i18n.global.t("DNOTE_CORRELATION_METRIC_NAME_DUPLICATED_ERROR")
            return
        }
        if (!Number.isInteger(Number(lag.value))) {
            validation_message.value = i18n.global.t("DNOTE_CORRELATION_LAG_INTEGER_ERROR")
            return
        }

        const query = new DnoteCorrelationGraphQuery()
        query.id = initial_query.value.id
        query.title = title.value
        query.granularity = granularity.value
        query.method = method.value
        query.lag = Number(lag.value)
        query.metrics = metrics.value.map(draft => {
            const metric = new DnoteCorrelationMetric()
            metric.id = draft.id
            metric.title = draft.title.trim()
            metric.aggregate_target = build_dnote_aggregate_target_from_json({ type: draft.aggregate_target })
            metric.predicate = build_dnote_predicate_from_json(predicate_struct_to_json(draft.root_predicate))
            // 集計対象を平均へ切り替えた後にチェックだけ残らないよう、保存時に落とす
            metric.missing_as_zero = draft.missing_as_zero && supports_missing_as_zero(draft)
            metric.timeis_span_policy = draft.timeis_span_policy
            return metric
        })
        validation_message.value = ""
        emits('saved', query)
    }

    // 開き直しや対象の差し替えで props が変わったら、編集中の値を捨てて読み直す
    watch(initial_query, load_query, { immediate: true })

    // ── Return ──
    return {
        // State
        title,
        granularity,
        method,
        lag,
        metrics,
        aggregate_targets,
        granularities,
        methods,
        timeis_span_policies,
        validation_message,

        // Business logic
        load_query,
        add_metric,
        supports_missing_as_zero,
        delete_metric,
        move_metric,
        save,
    }
}
