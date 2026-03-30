import { computed, type Ref, ref } from 'vue'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import { UpdateNotificationRequest } from '@/classes/api/req_res/update-notification-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { ConfirmDeleteNotificationViewProps } from '@/pages/views/confirm-delete-notification-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useConfirmDeleteNotificationView(options: {
    props: ConfirmDeleteNotificationViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const show_kyou: Ref<boolean> = ref(true)

    const notification_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifer = props.notification.generate_info_identifer()
        return [info_identifer]
    })

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

    // ── Delete logic ──
    async function delete_notification(): Promise<void> {
        // 更新後テキスト情報を用意する
        const updated_notification = props.notification.clone()
        updated_notification.is_deleted = true
        updated_notification.update_app = "gkill"
        updated_notification.update_device = props.application_config.device
        updated_notification.update_time = new Date(Date.now())
        updated_notification.update_user = props.application_config.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_notification.id)
        await delete_gkill_kyou_cache(updated_notification.target_id)
        const req = new UpdateNotificationRequest()
        req.notification = updated_notification
        const res = await props.gkill_api.update_notification(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits('deleted_notification', res.updated_notification)
        emits('requested_reload_kyou', props.kyou)
        emits('requested_close_dialog')
        return
    }

    // ── Return ──
    return {
        // State
        show_kyou,
        notification_highlight_targets,

        // Methods
        delete_notification,

        // Event relay objects
        crudRelayHandlers,
    }
}
