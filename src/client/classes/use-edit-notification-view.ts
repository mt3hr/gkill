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
        void cloned_notification.value.attached_histories[0]
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
