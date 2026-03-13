import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import { TimeIs } from '@/classes/datas/time-is'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { AddTimeisRequest } from '@/classes/api/req_res/add-timeis-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { EditTimeIsViewProps } from '@/pages/views/edit-time-is-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useAddTimeisView(options: {
    props: EditTimeIsViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)

    const timeis: Ref<TimeIs> = ref(((): TimeIs => {
        const timeis = new TimeIs()
        timeis.start_time = moment().toDate()
        timeis.end_time = null
        return timeis
    })())
    const timeis_title: Ref<string> = ref(timeis.value.title)
    const timeis_start_date_typed: Ref<Date> = ref(moment(timeis.value.start_time).toDate())
    const timeis_start_date_string: Ref<string> = computed(() => moment(timeis_start_date_typed.value).format("YYYY-MM-DD"))
    const timeis_start_time_string: Ref<string> = ref(moment(timeis.value.start_time).format("HH:mm:ss"))
    const timeis_end_date_typed: Ref<Date | null> = ref(null)
    const timeis_end_date_string: Ref<string> = computed(() => timeis_end_date_typed.value ? moment(timeis_end_date_typed.value).format("YYYY-MM-DD") : "")
    const timeis_end_time_string: Ref<string> = ref("")

    const show_start_date_menu = ref(false)
    const show_start_time_menu = ref(false)
    const show_end_date_menu = ref(false)
    const show_end_time_menu = ref(false)

    // ── Business logic ──
    function reset(): void {
        timeis_title.value = (timeis.value.title)
        timeis_start_date_string.value = (moment(timeis.value.start_time).format("YYYY-MM-DD"))
        timeis_start_time_string.value = (moment(timeis.value.start_time).format("HH:mm:ss"))
        timeis_end_date_typed.value = null
        timeis_end_time_string.value = ""
    }

    function reset_start_date_time(): void {
        timeis_start_date_typed.value = moment(timeis.value.start_time).toDate()
        timeis_start_time_string.value = moment(timeis.value.start_time).format("HH:mm:ss")
    }

    function reset_end_date_time(): void {
        timeis_end_date_typed.value = null
        timeis_end_time_string.value = ""
    }

    function now_to_start_date_time(): void {
        timeis_start_date_typed.value = moment().toDate()
        timeis_start_time_string.value = moment().format("HH:mm:ss")
    }

    function now_to_end_date_time(): void {
        timeis_end_date_typed.value = moment().toDate()
        timeis_end_time_string.value = moment().format("HH:mm:ss")
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            if (!timeis.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_timeis_is_null
                error.error_message = i18n.global.t("CLIENT_TIMEIS_IS_NULL_MESSAGE")
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

            // 終了日時 片方だけ入力されていたらエラーチェック
            if (timeis_end_date_string.value === "" || timeis_end_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
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

            // 更新後TimeIs情報を用意する
            let end_time: Date | null = null
            if (timeis_end_date_string.value !== "" && timeis_end_time_string.value !== "") {
                end_time = moment(timeis_end_date_string.value + " " + timeis_end_time_string.value).toDate()
            }
            const new_timeis = timeis.value.clone()
            new_timeis.id = props.gkill_api.generate_uuid()
            new_timeis.title = timeis_title.value
            new_timeis.start_time = moment(timeis_start_date_string.value + " " + timeis_start_time_string.value).toDate()
            new_timeis.end_time = end_time
            new_timeis.create_app = "gkill"
            new_timeis.create_device = props.application_config.device
            new_timeis.create_time = new Date(Date.now())
            new_timeis.create_user = props.application_config.user_id
            new_timeis.update_app = "gkill"
            new_timeis.update_device = props.application_config.device
            new_timeis.update_time = new Date(Date.now())
            new_timeis.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_timeis.id)
            const req = new AddTimeisRequest()
            req.timeis = new_timeis
            req.want_response_kyou = true
            const res = await props.gkill_api.add_timeis(req)
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

    // ── Return ──
    return {
        // State
        is_requested_submit,
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

        // Methods
        save,
        reset,
        reset_start_date_time,
        reset_end_date_time,
        now_to_start_date_time,
        now_to_end_date_time,
    }
}
