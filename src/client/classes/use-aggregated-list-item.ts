import { computed } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type AggregatedListItemProps from '@/pages/views/aggregated-list-item-props'
import type AggregatedListItemViewEmits from '@/pages/views/aggregated-list-item-view-emits'

export function useAggregatedListItem(options: {
    props: AggregatedListItemProps,
    emits: AggregatedListItemViewEmits,
}) {
    const { props, emits } = options

    // ── Computed ──
    const list_height = computed(() => window.screen.height * 7 / 10)

    const aggregate_target_type = computed(() => props.dnote_list_query.aggregate_target.to_json().type.toString())
    const is_lantana_type = computed(() => aggregate_target_type.value.includes("Lantana"))
    const is_plus_number_value = computed(() => {
        if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
            if (props.aggregated_item.value.toString().startsWith("-")) {
                return false
            } else {
                return true
            }
        }
        return false
    })
    const is_minus_number_value = computed(() => {
        if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
            if (props.aggregated_item.value.toString().startsWith("-")) {
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
    const crudRelayHandlers = {
        'received_errors': (...args: any[]) => emits('received_errors', args[0] as Array<GkillError>),
        'received_messages': (...args: any[]) => emits('received_messages', args[0] as Array<GkillMessage>),
        'focused_kyou': (...args: any[]) => emits('focused_kyou', args[0] as Kyou),
        'clicked_kyou': (...args: any[]) => { emits('focused_kyou', args[0] as Kyou); emits('clicked_kyou', args[0] as Kyou) },
        'deleted_kyou': (...args: any[]) => emits('deleted_kyou', args[0] as Kyou),
        'deleted_tag': (...args: any[]) => emits('deleted_tag', args[0] as Tag),
        'deleted_text': (...args: any[]) => emits('deleted_text', args[0] as Text),
        'deleted_notification': (...args: any[]) => emits('deleted_notification', args[0] as Notification),
        'registered_kyou': (...args: any[]) => emits('registered_kyou', args[0] as Kyou),
        'registered_tag': (...args: any[]) => emits('registered_tag', args[0] as Tag),
        'registered_text': (...args: any[]) => emits('registered_text', args[0] as Text),
        'registered_notification': (...args: any[]) => emits('registered_notification', args[0] as Notification),
        'updated_kyou': (...args: any[]) => emits('updated_kyou', args[0] as Kyou),
        'updated_tag': (...args: any[]) => emits('updated_tag', args[0] as Tag),
        'updated_text': (...args: any[]) => emits('updated_text', args[0] as Text),
        'updated_notification': (...args: any[]) => emits('updated_notification', args[0] as Notification),
        'requested_open_rykv_dialog': (...args: any[]) => emits('requested_open_rykv_dialog', args[0], args[1], args[2]),
    }

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
