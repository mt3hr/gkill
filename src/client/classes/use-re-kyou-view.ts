import { type Ref, ref, watch } from 'vue'
import type { ReKyouViewProps } from '@/pages/views/re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

export function useReKyouView(options: {
    props: ReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<any>(null)

    // ── State refs ──
    const target_kyou: Ref<Kyou> = ref(new Kyou())

    // ── Watchers ──
    watch(() => props.kyou, () => get_target_kyou())

    // ── Business logic ──
    async function get_target_kyou() {
        const req = new GetKyouRequest()
        req.id = props.rekyou.target_id
        const res = await props.gkill_api.get_kyou(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        target_kyou.value = res.kyou_histories[0]
    }

    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    // ── Init ──
    get_target_kyou()

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
        // Template refs
        context_menu,

        // State
        target_kyou,

        // Business logic
        show_context_menu,
        get_target_kyou,

        // Event relay objects
        crudRelayHandlers,
    }
}
