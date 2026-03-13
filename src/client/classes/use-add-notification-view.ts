import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { GkillError } from '@/classes/api/gkill-error'
import type { AddNotificationViewProps } from '@/pages/views/add-notification-view-props'
import { AddNotificationRequest } from '@/classes/api/req_res/add-notification-request'
import moment from 'moment'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import { Notification } from '@/classes/datas/notification'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'

export function useAddNotificationView(options: {
    props: AddNotificationViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(false)
    const content_value: Ref<string> = ref("")
    const notification_date_typed: Ref<Date> = ref(new Date(Date.now()))
    const notification_date_string: Ref<string> = computed(() => moment(notification_date_typed.value).format("YYYY-MM-DD"))
    const notification_time_string: Ref<string> = ref("")

    const show_notification_date_menu = ref(false)
    const show_notification_time_menu = ref(false)

    // ── Business logic ──
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

            // 通知内容情報を用意する
            const new_notification = new Notification()
            new_notification.notification_time = moment(notification_date_string.value + " " + notification_time_string.value).toDate()
            new_notification.content = content_value.value
            new_notification.id = props.gkill_api.generate_uuid()
            new_notification.is_deleted = false
            new_notification.target_id = props.kyou.id
            new_notification.related_time = new Date(Date.now())
            new_notification.create_app = "gkill"
            new_notification.create_device = props.application_config.device
            new_notification.create_time = new Date(Date.now())
            new_notification.create_user = props.application_config.user_id
            new_notification.update_app = "gkill"
            new_notification.update_device = props.application_config.device
            new_notification.update_time = new Date(Date.now())
            new_notification.update_user = props.application_config.user_id
            new_notification.related_time = new Date(Date.now())

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_notification.id)
            await delete_gkill_kyou_cache(new_notification.target_id)
            const req = new AddNotificationRequest()
            req.notification = new_notification
            const res = await props.gkill_api.add_notification(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function reset_notification_date_time(): void {
        notification_date_typed.value = new Date(Date.now())
        notification_time_string.value = ""
    }

    // ── Template event handlers ──
    function onCloseDateMenu(): void {
        show_notification_date_menu.value = false
    }

    function onCloseTimeMenu(): void {
        show_notification_time_menu.value = false
    }

    // ── CRUD relay handlers ──
    const kyouViewHandlers = {
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
    }

    return {
        // State
        is_requested_submit,
        show_kyou,
        content_value,
        notification_date_typed,
        notification_date_string,
        notification_time_string,
        show_notification_date_menu,
        show_notification_time_menu,

        // Business logic
        save,
        reset_notification_date_time,

        // Template event handlers
        onCloseDateMenu,
        onCloseTimeMenu,

        // Event relay objects
        kyouViewHandlers,
    }
}
