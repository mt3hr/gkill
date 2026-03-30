import { type Ref, ref } from 'vue'
import { ReKyou } from '@/classes/datas/re-kyou'
import { AddReKyouRequest } from '@/classes/api/req_res/add-re-kyou-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { ConfirmReKyouViewProps } from '@/pages/views/confirm-re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useConfirmReKyouView(options: {
    props: ConfirmReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const show_kyou: Ref<boolean> = ref(true)

    // ── CRUD relay handlers ──
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
    }

    // ── Rekyou logic ──
    async function rekyou(): Promise<void> {
        // rekyou情報を用意する
        const new_rekyou = new ReKyou()
        new_rekyou.id = props.gkill_api.generate_uuid()
        new_rekyou.is_deleted = false
        new_rekyou.target_id = props.kyou.id
        new_rekyou.related_time = new Date(Date.now())
        new_rekyou.create_app = "gkill"
        new_rekyou.create_device = props.application_config.device
        new_rekyou.create_time = new Date(Date.now())
        new_rekyou.create_user = props.application_config.user_id
        new_rekyou.update_app = "gkill"
        new_rekyou.update_device = props.application_config.device
        new_rekyou.update_time = new Date(Date.now())
        new_rekyou.update_user = props.application_config.user_id

        // 追加リクエストを飛ばす
        await delete_gkill_kyou_cache(new_rekyou.id)
        const req = new AddReKyouRequest()
        req.want_response_kyou = true
        req.rekyou = new_rekyou
        const res = await props.gkill_api.add_rekyou(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits('requested_close_dialog')
        return
    }

    // ── Return ──
    return {
        // State
        show_kyou,

        // Methods
        rekyou,

        // Event relay objects
        crudRelayHandlers,
    }
}
