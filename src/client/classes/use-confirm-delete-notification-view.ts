import { computed, type Ref, ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import { UpdateNotificationRequest } from '@/classes/api/req_res/update-notification-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ConfirmDeleteNotificationViewProps } from '@/pages/views/confirm-delete-notification-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useConfirmDeleteNotificationView(options: {
    props: ConfirmDeleteNotificationViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(true)

    const notification_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifier = props.notification.generate_info_identifier()
        return [info_identifier]
    })

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Delete logic ──
    //
    // 何があってもダイアログを閉じる。理由は use-confirm-delete-kyou-view.ts と同じ
    async function delete_notification(): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            // 更新後通知情報を用意する
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
        } catch (err: unknown) {
            console.error(err)
            const error = new GkillError()
            error.error_code = GkillErrorCodes.failed_delete_notification
            error.error_message = i18n.global.t("FAILED_UPDATE_NOTIFICATION_MESSAGE")
            emits('received_errors', [error])
        } finally {
            is_requested_submit.value = false
            emits('requested_close_dialog')
        }
    }

    // ── Return ──
    return {
        // State
        is_requested_submit,
        show_kyou,
        notification_highlight_targets,

        // Methods
        delete_notification,

        // Event relay objects
        crudRelayHandlers,
    }
}

