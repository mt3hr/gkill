import { ref } from 'vue'
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
        'deleted_kyou': (...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou),
        'deleted_tag': (...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag),
        'deleted_text': (...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text),
        'deleted_notification': (...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification),
        'registered_kyou': (...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou),
        'registered_tag': (...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag),
        'registered_text': (...registered_text: any[]) => emits('registered_text', registered_text[0] as Text),
        'registered_notification': (...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification),
        'updated_kyou': (...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou),
        'updated_tag': (...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag),
        'updated_text': (...updated_text: any[]) => emits('updated_text', updated_text[0] as Text),
        'updated_notification': (...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification),
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
        'requested_reload_kyou': (...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean),
        'requested_open_rykv_dialog': (...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2]),
    }

    // ── Return ──
    return {
        // State
        cloned_kyou,

        // Event relay objects
        crudRelayHandlers,
    }
}
