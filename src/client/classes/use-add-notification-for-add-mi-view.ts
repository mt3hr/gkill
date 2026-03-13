import { computed, ref, type Ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { Notification } from '@/classes/datas/notification'
import moment from 'moment'
import type { AddNotificationForAddMiViewProps } from '@/pages/views/add-notification-for-add-mi-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useAddNotificationForAddMiView(options: {
    props: AddNotificationForAddMiViewProps,
    emits: KyouViewEmits
}) {
    const props = options.props
    const emits = options.emits

    // ── State refs ──
    const is_requested_submit = ref(false)
    const content_value: Ref<string> = ref(props.default_notification ? props.default_notification.content : "")
    const notification_date_typed: Ref<Date> = ref(props.default_notification && props.default_notification.notification_time.getTime() !== new Date(0).getTime() ? moment(props.default_notification.notification_time).toDate() : new Date(Date.now()))
    const notification_date_string: Ref<string> = computed(() => moment(notification_date_typed.value).format("YYYY-MM-DD"))
    const notification_time_string: Ref<string> = ref(props.default_notification && props.default_notification.notification_time.getTime() !== new Date(0).getTime() ? moment(props.default_notification.notification_time).format("HH:mm:ss") : "")

    const show_notification_date_menu = ref(false)
    const show_notification_time_menu = ref(false)

    // ── Methods ──
    async function get_notification(): Promise<Notification | null> {
        // 値がなかったらエラーメッセージを出力する
        if (content_value.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.notification_content_is_blank
            error.error_message = i18n.global.t("NOTIFICATION_CONTENT_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return null
        }
        // 通知日時 入力なしエラーチェック
        if (notification_date_string.value === "" || notification_time_string.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.notification_time_is_blank
            error.error_message = i18n.global.t("NOTIFICATION_DATE_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return null
        }

        // 通知情報を用意する
        const new_notification = new Notification()
        new_notification.notification_time = moment(notification_date_string.value + " " + notification_time_string.value).toDate()
        new_notification.content = content_value.value
        new_notification.id = props.default_notification ? props.default_notification.id : props.gkill_api.generate_uuid()
        new_notification.is_deleted = false
        new_notification.target_id = props.kyou.id
        new_notification.related_time = new Date(Date.now())
        new_notification.create_app = "gkill"
        new_notification.create_device = props.application_config.device
        new_notification.create_time = new Date(Date.now())
        new_notification.create_user = props.application_config.user_id
        new_notification.update_app = "gkill"
        new_notification.update_app = "gkill"
        new_notification.update_device = props.application_config.device
        new_notification.update_time = new Date(Date.now())
        new_notification.update_user = props.application_config.user_id
        new_notification.related_time = new Date(Date.now())
        return new_notification
    }

    function reset_notification_date_time(): void {
        notification_date_typed.value = props.default_notification && props.default_notification.notification_time.getTime() !== new Date(0).getTime() ? moment(props.default_notification.notification_time).toDate() : new Date(Date.now())
        notification_time_string.value = props.default_notification && props.default_notification.notification_time.getTime() !== new Date(0).getTime() ? moment(props.default_notification.notification_time).format("HH:mm:ss") : ""
    }

    return {
        // State
        is_requested_submit,
        content_value,
        notification_date_typed,
        notification_date_string,
        notification_time_string,
        show_notification_date_menu,
        show_notification_time_menu,

        // Methods
        get_notification,
        reset_notification_date_time,
    }
}
