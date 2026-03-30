import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { EndTimeIsPlaingViewProps } from '@/pages/views/end-time-is-plaing-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateTimeisRequest } from '@/classes/api/req_res/update-timeis-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'

export function useEndTimeIsPlaingView(options: {
    props: EndTimeIsPlaingViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)

    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const timeis_title: Ref<string> = ref(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : "")
    const timeis_start_date_typed: Ref<Date> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.start_time).toDate() : new Date(0))
    const timeis_start_date_string: Ref<string> = computed(() => moment(timeis_start_date_typed.value).format("YYYY-MM-DD"))
    const timeis_start_time_string: Ref<string> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.start_time).format("HH:mm:ss") : "")
    const timeis_end_date_typed: Ref<Date | null> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.end_time).toDate() : null)
    const timeis_end_date_string: Ref<string> = computed(() => timeis_end_date_typed.value ? moment(timeis_end_date_typed.value).format("YYYY-MM-DD") : "")
    const timeis_end_time_string: Ref<string> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.end_time).format("HH:mm:ss") : "")

    const show_start_date_menu = ref(false)
    const show_start_time_menu = ref(false)
    const show_end_date_menu = ref(false)
    const show_end_time_menu = ref(false)

    const show_kyou: Ref<boolean> = ref(false)

    // ── Watchers ──
    watch(() => props.kyou, () => load())

    // ── Business logic ──
    async function load(): Promise<void> {
        cloned_kyou.value = props.kyou.clone()
        await cloned_kyou.value.reload(false, true)
        await cloned_kyou.value.load_typed_datas()
        await cloned_kyou.value.load_all()
        timeis_title.value = cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : ""
        timeis_start_date_typed.value = moment(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.start_time : "").toDate()
        timeis_start_time_string.value = moment(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.start_time : "").format("HH:mm:ss")
        timeis_end_date_typed.value = moment().toDate()
        timeis_end_time_string.value = moment().format("HH:mm:ss")
    }

    function reset(): void {
        timeis_title.value = cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : ""
        reset_start_date_time()
        reset_end_date_time()
    }

    function reset_start_date_time(): void {
        timeis_start_date_typed.value = cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis.start_time).toDate() : new Date(Date.now())
        timeis_start_time_string.value = cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis.start_time).format("HH:mm:ss") : ""
    }

    function reset_end_date_time(): void {
        timeis_end_date_typed.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).toDate() : null
        timeis_end_time_string.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).format("HH:mm:ss") : ""
    }

    function clear_end_date_time(): void {
        timeis_end_date_typed.value = null
        timeis_end_time_string.value = ""
    }

    function now_to_end_date_time(): void {
        timeis_end_date_typed.value = moment().toDate()
        timeis_end_time_string.value = moment().format("HH:mm:ss")
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const timeis = cloned_kyou.value.typed_timeis
            if (!timeis) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_timeis_is_null
                error.error_message = i18n.global.t("CLIENT_TIMEIS_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タイトル入力チェック
            if (timeis_title.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.timeis_title_is_blank
                error.error_message = i18n.global.t("TIMEIS_TITLE_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 開始日時必須入力チェック
            if (timeis_start_date_string.value === "" || timeis_start_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.timeis_start_time_is_blank
                error.error_message = i18n.global.t("TIMEIS_START_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 終了日時入力チェック
            if ((timeis_end_date_string.value === "" && timeis_end_time_string.value !== "") ||
                (timeis_end_date_string.value !== "" && timeis_end_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                const error = new GkillError()
                error.error_code = GkillErrorCodes.timeis_end_time_is_blank
                error.error_message = i18n.global.t("TIMEIS_END_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する
            if (timeis.title === timeis_title.value &&
                (moment(timeis.start_time).toDate().getTime() === moment(timeis_start_date_string.value + " " + timeis_start_time_string.value).toDate().getTime()) &&
                (moment(timeis.end_time).toDate().getTime() === moment(timeis_end_date_string.value + " " + timeis_end_time_string.value).toDate().getTime()) || (timeis.end_time === null && timeis_end_date_string.value === "" && timeis_end_time_string.value === "")) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.timeis_is_no_update
                error.error_message = i18n.global.t("TIMEIS_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後TimeIs情報を用意する
            let end_time: Date | null = null
            if (timeis_end_date_string.value !== "" && timeis_end_time_string.value !== "") {
                end_time = moment(timeis_end_date_string.value + " " + timeis_end_time_string.value).toDate()
            }
            const updated_timeis = timeis.clone()
            updated_timeis.title = timeis_title.value
            updated_timeis.start_time = moment(timeis_start_date_string.value + " " + timeis_start_time_string.value).toDate()
            updated_timeis.end_time = end_time
            updated_timeis.update_app = "gkill"
            updated_timeis.update_device = props.application_config.device
            updated_timeis.update_time = new Date(Date.now())
            updated_timeis.update_user = props.application_config.user_id

            // 更新リクエストを飛ばす
            await delete_gkill_kyou_cache(updated_timeis.id)
            const req = new UpdateTimeisRequest()
            req.timeis = updated_timeis
            req.want_response_kyou = true
            const res = await props.gkill_api.update_timeis(req)
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
        timeis_title,
        timeis_start_date_typed,
        timeis_start_date_string,
        timeis_start_time_string,
        timeis_end_date_typed,
        timeis_end_date_string,
        timeis_end_time_string,
        show_start_date_menu,
        show_start_time_menu,
        show_end_date_menu,
        show_end_time_menu,
        show_kyou,

        // Business logic
        reset,
        reset_end_date_time,
        clear_end_date_time,
        now_to_end_date_time,
        save,

        // Event relay objects
        crudRelayHandlers,
    }
}
