import { ref } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { KyouHistoriesViewProps } from '@/pages/views/kyou-histories-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

export function useKyouHistoriesView(options: {
    props: KyouHistoriesViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const cloned_kyou = ref(new Kyou())

    // ── Init ──
    load_cloned_kyou()

    // ── Business logic ──
    async function load_cloned_kyou() {
        const cloned_kyou_value = props.kyou.clone()
        await cloned_kyou_value.load_attached_histories()
        for (let i = 0; i < cloned_kyou.value.attached_histories.length; i++) {
            cloned_kyou_value.attached_histories[i].related_time = cloned_kyou_value.attached_histories[i].update_time
        }
        cloned_kyou.value = cloned_kyou_value
    }

    // ── Event relay objects ──
    const crudRelayHandlers = {
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
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_reload_kyou': (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => emits('requested_update_check_kyous', kyous, checked),
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    // ── Return ──
    return {
        // State
        cloned_kyou,

        // Event relay objects
        crudRelayHandlers,
    }
}
