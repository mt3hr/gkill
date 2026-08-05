'use strict'

import { nextTick, ref, type Ref } from 'vue'
import type PredicateGroupType from '@/classes/dnote/predicate-group-type'
import type Predicate from '@/classes/dnote/predicate'
import aggregate_target_menu_items from '@/classes/dnote/pulldown-menu/aggregate-target-menu-items'
import trend_granularity_menu_items from '@/classes/dnote/pulldown-menu/trend-granularity-menu-items'
import trend_chart_type_menu_items from '@/classes/dnote/pulldown-menu/trend-chart-type-menu-items'
import type DnoteSelectItem from '@/classes/dnote/dnote-select-item'
import DnoteTrendGraphQuery from '@/pages/views/dnote-trend-graph-query'
import type { DnoteTrendChartType, DnoteTrendGranularity } from '@/classes/dnote/dnote-trend/dnote-trend-types'
import type EditDnoteTrendGraphViewEmits from '@/pages/views/edit-dnote-trend-graph-view-emits'
import type EditDnoteTrendGraphViewProps from '@/pages/views/edit-dnote-trend-graph-view-props'
import { build_dnote_aggregate_target_from_json, build_dnote_predicate_from_json } from '@/classes/dnote/serialize/register-dictionary'

export function useEditDnoteTrendGraphView(options: {
    props: EditDnoteTrendGraphViewProps
    emits: EditDnoteTrendGraphViewEmits
}) {
    const { props, emits } = options

    const id = ref(props.gkill_api.generate_uuid())
    const title = ref("")

    const root_predicate = ref<PredicateGroupType>({
        logic: 'AND',
        predicates: []
    })

    const aggregate_targets: Ref<Array<DnoteSelectItem>> = ref(aggregate_target_menu_items)
    const aggregate_target: Ref<string> = ref(aggregate_targets.value[0].value)

    const granularities: Ref<Array<DnoteSelectItem>> = ref(trend_granularity_menu_items)
    const granularity: Ref<string> = ref(granularities.value[0].value)

    const chart_types: Ref<Array<DnoteSelectItem>> = ref(trend_chart_type_menu_items)
    const chart_type: Ref<string> = ref(chart_types.value[0].value)

    nextTick(() => load_props())

    async function load_props(): Promise<void> {
        id.value = props.dnote_trend_graph_query.id
        title.value = props.dnote_trend_graph_query.title
        root_predicate.value = predicate_struct_from_json(props.dnote_trend_graph_query.predicate.predicate_struct_to_json()) as PredicateGroupType
        aggregate_target.value = aggregate_targets.value.find((aggregate_target) => aggregate_target.value === props.dnote_trend_graph_query.aggregate_target.to_json().type)!.value
        granularity.value = props.dnote_trend_graph_query.granularity
        chart_type.value = props.dnote_trend_graph_query.chart_type
    }

    async function reset(): Promise<void> {
        return load_props()
    }

    async function save(): Promise<void> {
        const new_dnote_trend_graph_query = new DnoteTrendGraphQuery()
        new_dnote_trend_graph_query.id = id.value
        new_dnote_trend_graph_query.title = title.value
        new_dnote_trend_graph_query.aggregate_target = build_dnote_aggregate_target_from_json({ type: aggregate_target.value })
        new_dnote_trend_graph_query.predicate = build_dnote_predicate_from_json(predicate_struct_to_json(root_predicate.value))
        new_dnote_trend_graph_query.granularity = granularity.value as DnoteTrendGranularity
        new_dnote_trend_graph_query.chart_type = chart_type.value as DnoteTrendChartType

        emits('requested_update_dnote_trend_graph', new_dnote_trend_graph_query)
        emits('requested_close_dialog')
    }

    function predicate_struct_to_json(group: PredicateGroupType | Predicate): Record<string, unknown> {
        if (is_group(group)) {
            return {
                logic: group.logic,
                predicates: group.predicates.map(p => predicate_struct_to_json(p))
            }
        } else {
            return { type: group.type, value: group.value }
        }
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    function predicate_struct_from_json(json: any): PredicateGroupType | Predicate {
        if (json.logic && Array.isArray(json.predicates)) {
            return {
                logic: json.logic,
                predicates: json.predicates.map(predicate_struct_from_json)
            }
        } else {
            return {
                type: json.type,
                value: json.value
            }
        }
    }

    function is_group(p: Predicate | PredicateGroupType): p is PredicateGroupType {
        return 'logic' in p && Array.isArray(p.predicates)
    }

    return {
        id,
        title,
        root_predicate,
        aggregate_targets,
        aggregate_target,
        granularities,
        granularity,
        chart_types,
        chart_type,
        reset,
        save,
    }
}
