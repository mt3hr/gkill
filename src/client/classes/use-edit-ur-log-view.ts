import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateURLogRequest } from '@/classes/api/req_res/update-ur-log-request'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { EditURLogViewProps } from '@/pages/views/edit-ur-log-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'

export function useEditUrLogView(options: {
    props: EditURLogViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const title: Ref<string> = ref(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.title : "")
    const url: Ref<string> = ref(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.url : "")
    const related_date_typed: Ref<Date> = ref(moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").format("HH:mm:ss"))
    const re_get_urlog_content: Ref<boolean> = ref(true)
    const show_kyou: Ref<boolean> = ref(false)
    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Watchers ──
    watch(() => props.kyou, () => load())

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

    // ── Business logic ──
    async function load(): Promise<void> {
        cloned_kyou.value = props.kyou.clone()
        await cloned_kyou.value.reload(false, true)
        await cloned_kyou.value.load_typed_datas()
        await cloned_kyou.value.load_all()
        title.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.title : ""
        url.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.url : ""
        related_date_typed.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").toDate()
        related_time_string.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").format("HH:mm:ss")
    }

    function reset(): void {
        title.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.title : ""
        url.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.url : ""
        related_date_typed.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").toDate()
        related_time_string.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").format("HH:mm:ss")
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const urlog = cloned_kyou.value.typed_urlog
            if (!urlog) {
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

            // 更新がなかったらエラーメッセージを出力する
            if (urlog.title === title.value &&
                urlog.url === url.value &&
                moment(urlog.related_time).toDate().getTime() === moment(related_date_string.value + " " + related_time_string.value).toDate().getTime()) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.urlog_is_no_update
                error.error_message = i18n.global.t("URLOG_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後URLog情報を用意する
            const updated_urlog = urlog.clone()
            updated_urlog.title = title.value
            updated_urlog.url = url.value
            updated_urlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            updated_urlog.update_app = "gkill"
            updated_urlog.update_device = props.application_config.device
            updated_urlog.update_time = new Date(Date.now())
            updated_urlog.update_user = props.application_config.user_id

            // 再取得の場合、URLとタイトル以外をブランクにする
            if (re_get_urlog_content.value) {
                updated_urlog.description = ""
                updated_urlog.favicon_image = ""
                updated_urlog.thumbnail_image = ""
            }

            // 更新リクエストを飛ばす
            await delete_gkill_kyou_cache(updated_urlog.id)
            const req = new UpdateURLogRequest()
            req.urlog = updated_urlog
            req.want_response_kyou = true
            const res = await props.gkill_api.update_urlog(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits("updated_kyou", res.updated_kyou!)
            emits('requested_reload_kyou', props.kyou)
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
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
    }

    // ── Initialize ──
    load()

    // ── Return ──
    return {
        // State
        is_requested_submit,
        cloned_kyou,
        title,
        url,
        related_date_typed,
        related_date_string,
        related_time_string,
        re_get_urlog_content,
        show_kyou,
        show_related_date_menu,
        show_related_time_menu,

        // Methods
        save,
        reset,
        now_to_related_date_time,
        reset_related_date_time,

        // Event relay objects
        crudRelayHandlers,
    }
}
