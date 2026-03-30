import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import { GkillError } from '@/classes/api/gkill-error'
import moment from 'moment'
import { UpdateNlogRequest } from '@/classes/api/req_res/update-nlog-request'
import type { EditNlogViewProps } from '@/pages/views/edit-nlog-view-props'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'

export function useEditNlogView(options: {
    props: EditNlogViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_loading = ref(true)
    const is_requested_submit = ref(false)
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const nlog_title_value: Ref<string> = ref(cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.title : "")
    const nlog_amount_value: Ref<number> = ref(cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.amount : 0)
    const nlog_shop_value: Ref<string> = ref(cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.shop : "")
    const related_date_typed: Ref<Date> = ref(moment(cloned_kyou.value.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(cloned_kyou.value.related_time).format("HH:mm:ss"))
    const show_kyou: Ref<boolean> = ref(false)
    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Watchers ──
    watch(() => props.kyou, () => load())

    // ── CRUD relay handlers ──
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

    // ── Business logic ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.reload(false, true)
            await cloned_kyou.value.load_typed_datas()
            await cloned_kyou.value.load_all()
            nlog_title_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.title : ""
            nlog_amount_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.amount : 0
            nlog_shop_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.shop : ""
            related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
            related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
        } finally {
            is_loading.value = false
        }
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const nlog = cloned_kyou.value.typed_nlog
            if (!nlog) {
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
            if (Number.isNaN(nlog_amount_value.value)) {
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

            // 更新がなかったらエラーメッセージを出力する
            if (nlog_amount_value.value === nlog.amount &&
                nlog_shop_value.value === nlog.shop &&
                nlog_title_value.value === nlog.title &&
                moment(related_date_string.value + " " + related_time_string.value).toDate().getTime() === moment(nlog.related_time).toDate().getTime()) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.nlog_is_no_update
                error.error_message = i18n.global.t("NLOG_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後Nlog情報を用意する
            const updated_nlog = nlog.clone()
            updated_nlog.amount = nlog_amount_value.value
            updated_nlog.shop = nlog_shop_value.value
            updated_nlog.title = nlog_title_value.value
            updated_nlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            updated_nlog.update_app = "gkill"
            updated_nlog.update_device = props.application_config.device
            updated_nlog.update_time = new Date(Date.now())
            updated_nlog.update_user = props.application_config.user_id

            // 更新リクエストを飛ばす
            await delete_gkill_kyou_cache(updated_nlog.id)
            const req = new UpdateNlogRequest()
            req.want_response_kyou = true
            req.nlog = updated_nlog

            const res = await props.gkill_api.update_nlog(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits('updated_kyou', res.updated_kyou!)
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

    function reset(): void {
        nlog_title_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.title : ""
        nlog_amount_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.amount : 0
        nlog_shop_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.shop : ""
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
    }

    // ── Initialize ──
    load()

    // ── Return ──
    return {
        // State
        is_loading,
        is_requested_submit,
        cloned_kyou,
        nlog_title_value,
        nlog_amount_value,
        nlog_shop_value,
        related_date_typed,
        related_date_string,
        related_time_string,
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
