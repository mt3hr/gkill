import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { EditNotificationViewProps } from '@/pages/views/edit-notification-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateNotificationRequest } from '@/classes/api/req_res/update-notification-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'

export function useEditNotificationView(options: {
    props: EditNotificationViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)

    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const cloned_notification: Ref<Notification> = ref(props.notification.clone())
    const content_value: Ref<string> = ref(cloned_notification.value.content)
    const notification_date_typed: Ref<Date> = ref(moment(props.notification.notification_time).toDate())
    const notification_date_string: Ref<string> = computed(() => moment(notification_date_typed.value).format("YYYY-MM-DD"))
    const notification_time_string: Ref<string> = ref(moment(props.notification.notification_time).format("HH:mm:ss"))
    const show_kyou: Ref<boolean> = ref(false)

    const show_notification_date_menu = ref(false)
    const show_notification_time_menu = ref(false)

    // ── Watchers ──
    watch([() => props.kyou, () => props.notification], () => load())

    // ── Business logic ──
    async function load(): Promise<void> {
        cloned_kyou.value = props.kyou.clone()
        await cloned_kyou.value.reload(false, true)
        await cloned_kyou.value.load_typed_datas()
        await cloned_kyou.value.load_all()
        cloned_notification.value = props.notification.clone()
        cloned_notification.value.attached_histories[0]
        content_value.value = cloned_notification.value.content
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // 日時必須入力チェック
            if (notification_date_string.value === "" || notification_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.notification_time_is_blank
                error.error_message = i18n.global.t("NOTIFICATION_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 値がなかったらエラーメッセージを出力する
            if (content_value.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.notification_content_is_blank
                error.error_message = i18n.global.t("NOTIFICATION_CONTENT_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する
            if (cloned_notification.value.content === content_value.value &&
                (moment(cloned_notification.value.notification_time).toDate().getTime() === moment(notification_date_string.value + " " + notification_time_string.value).toDate().getTime())) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.notification_is_no_update
                error.error_message = i18n.global.t("NOTIFICATION_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後通知情報を用意する
            const updated_notification = cloned_notification.value.clone()
            updated_notification.content = content_value.value
            updated_notification.notification_time = moment(notification_date_string.value + " " + notification_time_string.value).toDate()
            updated_notification.update_app = "gkill"
            updated_notification.update_device = props.application_config.device
            updated_notification.update_time = moment(new Date(Date.now())).toDate()
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
            emits("updated_notification", res.updated_notification)
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function reset(): void {
        content_value.value = props.notification.content
        notification_date_typed.value = moment(props.notification.notification_time).toDate()
        notification_time_string.value = moment(props.notification.notification_time).format("HH:mm:ss")
    }

    function reset_notification_date_time(): void {
        notification_date_typed.value = moment(props.notification.notification_time).toDate()
        notification_time_string.value = moment(props.notification.notification_time).format("HH:mm:ss")
    }

    // ── Event relay objects ──
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

    // ── Init calls ──
    load()

    // ── Return ──
    return {
        // State
        is_requested_submit,
        cloned_kyou,
        cloned_notification,
        content_value,
        notification_date_typed,
        notification_date_string,
        notification_time_string,
        show_kyou,
        show_notification_date_menu,
        show_notification_time_menu,

        // Business logic
        reset,
        reset_notification_date_time,
        save,

        // Event relay objects
        crudRelayHandlers,
    }
}
