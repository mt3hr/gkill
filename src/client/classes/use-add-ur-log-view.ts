import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditURLogViewProps } from '@/pages/views/edit-ur-log-view-props'
import { URLog } from '@/classes/datas/ur-log'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { AddURLogRequest } from '@/classes/api/req_res/add-ur-log-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

export function useAddUrlogView(options: {
    props: EditURLogViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)

    const urlog: Ref<URLog> = ref((() => {
        const urlog = new URLog()
        urlog.related_time = new Date(Date.now())
        return urlog
    })())
    const title: Ref<string> = ref(urlog.value.title)
    const url: Ref<string> = ref(urlog.value.url)
    const related_date_typed: Ref<Date> = ref(moment(urlog.value.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(urlog.value.related_time).format("HH:mm:ss"))

    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Business logic ──
    function reset(): void {
        title.value = urlog.value.title
        url.value = urlog.value.url
        related_date_typed.value = moment(urlog.value.related_time).toDate()
        related_time_string.value = moment(urlog.value.related_time).format("HH:mm:ss")
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            if (!urlog.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_urlog_is_null
                error.error_message = i18n.global.t("CLIENT_URLOG_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.urlog_related_time_is_blank
                error.error_message = i18n.global.t("URLOG_DATE_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // URL入力チェック
            if (url.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.urlog_url_is_blank
                error.error_message = i18n.global.t("URLOG_URL_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後URLog情報を用意する
            const new_urlog = urlog.value.clone()
            new_urlog.id = props.gkill_api.generate_uuid()
            new_urlog.title = title.value
            new_urlog.url = url.value
            new_urlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            new_urlog.create_app = "gkill"
            new_urlog.create_device = props.application_config.device
            new_urlog.create_time = new Date(Date.now())
            new_urlog.create_user = props.application_config.user_id
            new_urlog.update_app = "gkill"
            new_urlog.update_device = props.application_config.device
            new_urlog.update_time = new Date(Date.now())
            new_urlog.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_urlog.id)
            const req = new AddURLogRequest()
            req.urlog = new_urlog
            req.want_response_kyou = true
            const res = await props.gkill_api.add_urlog(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits("registered_kyou", res.added_kyou!)
            emits('requested_reload_list')
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function now_to_related_date_time(): void {
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    function reset_related_date_time(): void {
        related_date_typed.value = moment(urlog.value.related_time).toDate()
        related_time_string.value = moment(urlog.value.related_time).format("HH:mm:ss")
    }

    // ── Template event handlers ──
    function onCloseDateMenu(): void {
        show_related_date_menu.value = false
    }

    function onCloseTimeMenu(): void {
        show_related_time_menu.value = false
    }

    return {
        // State
        is_requested_submit,
        title,
        url,
        related_date_typed,
        related_date_string,
        related_time_string,
        show_related_date_menu,
        show_related_time_menu,

        // Business logic
        reset,
        save,
        now_to_related_date_time,
        reset_related_date_time,

        // Template event handlers
        onCloseDateMenu,
        onCloseTimeMenu,
    }
}
