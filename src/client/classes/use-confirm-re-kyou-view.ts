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
        'received_errors': (...args: any[]) => emits('received_errors', args[0] as Array<GkillError>),
        'received_messages': (...args: any[]) => emits('received_messages', args[0] as Array<GkillMessage>),
        'requested_reload_kyou': (...args: any[]) => emits('requested_reload_kyou', args[0] as Kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (...args: any[]) => emits('requested_update_check_kyous', args[0] as Array<Kyou>, args[1] as boolean),
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
