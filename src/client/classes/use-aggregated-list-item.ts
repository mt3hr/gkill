import { computed } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'
import type AggregatedListItemProps from '@/pages/views/aggregated-list-item-props'
import type AggregatedListItemViewEmits from '@/pages/views/aggregated-list-item-view-emits'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'

export function useAggregatedListItem(options: {
    props: AggregatedListItemProps,
    emits: AggregatedListItemViewEmits,
}) {
    const { props, emits } = options

    // ── Computed ──
    const list_height = computed(() => window.screen.height * 7 / 10)

    const aggregate_target_type = computed(() => (props.dnote_list_query.aggregate_target.to_json().type as string).toString())
    const is_lantana_type = computed(() => aggregate_target_type.value.includes("Lantana"))
    const is_plus_number_value = computed(() => {
        if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
            if (String(props.aggregated_item.value).startsWith("-")) {
                return false
            } else {
                return true
            }
        }
        return false
    })
    const is_minus_number_value = computed(() => {
        if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
            if (String(props.aggregated_item.value).startsWith("-")) {
                return true
            }
        }
        return false
    })
    const value_class = computed(() => {
        if (is_plus_number_value.value) {
            return "plus_value"
        } else if (is_minus_number_value.value) {
            return "minus_value"
        }
        return ""
    })
    const mood_value = computed(() => Number(props.aggregated_item.value).valueOf())

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        // クリックはフォーカス移動も伴う
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })

    // ── Return ──
    return {
        // State
        list_height,
        is_lantana_type,
        value_class,
        mood_value,

        // Event relay objects
        crudRelayHandlers,
    }
}
