import { computed } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
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
    const crudRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'focused_kyou': (kyou: Kyou) => emits('focused_kyou', kyou),
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
        'deleted_kyou': (kyou: Kyou) => emits('deleted_kyou', kyou),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => emits('registered_kyou', kyou),
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => emits('updated_kyou', kyou),
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
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
