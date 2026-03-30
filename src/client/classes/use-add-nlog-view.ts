import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import { GkillError } from '@/classes/api/gkill-error'
import moment from 'moment'
import { Nlog } from '@/classes/datas/nlog'
import { AddNlogRequest } from '@/classes/api/req_res/add-nlog-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { AddNlogViewProps } from '@/pages/views/add-nlog-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useAddNlogView(options: {
    props: AddNlogViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)

    const nlog: Ref<Nlog> = ref((() => {
        const nlog = new Nlog()
        nlog.related_time = new Date(Date.now())
        return nlog
    })())
    const nlog_title_value: Ref<string> = ref("")
    const nlog_amount_value: Ref<number> = ref(0)
    const nlog_shop_value: Ref<string> = ref("")

    const related_date_typed: Ref<Date> = ref(moment().toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment().format("HH:mm:ss"))

    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Business logic ──
    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            if (!nlog.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_nlog_is_null
                error.error_message = i18n.global.t("CLIENT_NLOG_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.nlog_related_time_is_blank
                error.error_message = i18n.global.t("NLOG_DATE_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 金額入力チェック
            if (Number.isNaN(nlog_amount_value.value) || nlog_amount_value.value.toString() === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.nlog_amount_is_blank
                error.error_message = i18n.global.t("NLOG_AMOUNT_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 店名入力チェック
            if (nlog_shop_value.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.nlog_shop_name_is_blank
                error.error_message = i18n.global.t("NLOG_SHOP_NAME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タイトル入力チェック
            if (nlog_title_value.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.nlog_title_is_blank
                error.error_message = i18n.global.t("NLOG_TITLE_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後Nlog情報を用意する
            const new_nlog = nlog.value.clone()
            new_nlog.id = props.gkill_api.generate_uuid()
            new_nlog.amount = nlog_amount_value.value
            new_nlog.shop = nlog_shop_value.value
            new_nlog.title = nlog_title_value.value
            new_nlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            new_nlog.create_app = "gkill"
            new_nlog.create_device = props.application_config.device
            new_nlog.create_time = new Date(Date.now())
            new_nlog.create_user = props.application_config.user_id
            new_nlog.update_app = "gkill"
            new_nlog.update_device = props.application_config.device
            new_nlog.update_time = new Date(Date.now())
            new_nlog.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_nlog.id)
            const req = new AddNlogRequest()
            req.want_response_kyou = true
            req.nlog = new_nlog
            const res = await props.gkill_api.add_nlog(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits('registered_kyou', res.added_kyou!)
            emits('requested_reload_list')
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function reset_related_date_time(): void {
        related_date_typed.value = moment(nlog.value.related_time).toDate()
        related_time_string.value = moment(nlog.value.related_time).format("HH:mm:ss")
    }

    function now_to_related_date_time(): void {
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    function reset(): void {
        nlog_title_value.value = (nlog.value ? nlog.value.title : "")
        nlog_amount_value.value = (nlog.value ? nlog.value.amount : 0)
        nlog_shop_value.value = (nlog.value ? nlog.value.shop : "")
        related_date_typed.value = (moment().toDate())
        related_time_string.value = (moment().format("HH:mm:ss"))
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Return ──
    return {
        // State
        is_requested_submit,
        nlog,
        nlog_title_value,
        nlog_amount_value,
        nlog_shop_value,
        related_date_typed,
        related_date_string,
        related_time_string,
        show_related_date_menu,
        show_related_time_menu,

        // Business logic / template handlers
        save,
        reset_related_date_time,
        now_to_related_date_time,
        reset,

        // Event relay objects
        crudRelayHandlers,
    }
}
