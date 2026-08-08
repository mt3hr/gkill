import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditKCViewProps } from '@/pages/views/edit-kc-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateKCRequest } from '@/classes/api/req_res/update-kc-request'
import moment from 'moment'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useEditKCView(options: {
    props: EditKCViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_loading = ref(true)
    const is_requested_submit = ref(false)
    const is_busy = computed(() => is_loading.value || is_requested_submit.value)
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const title: Ref<string> = ref(cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.title : "")
    const num_value: Ref<number> = ref(cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.num_value : 0)
    const related_date_typed: Ref<Date> = ref(moment(cloned_kyou.value.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(cloned_kyou.value.related_time).format("HH:mm:ss"))
    const show_kyou: Ref<boolean> = ref(false)
    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Business logic ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.load_typed_datas()
            title.value = cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.title : ""
            num_value.value = cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.num_value : 0
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
            const kc = cloned_kyou.value.typed_kc
            if (!kc) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_kc_is_null
                error.error_message = i18n.global.t("CLIENT_KC_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タイトル入力チェック
            if (title.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kc_title_is_blank
                error.error_message = i18n.global.t("KC_TITLE_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 数値入力チェック
            if (num_value.value === null) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kc_num_value_is_blank
                error.error_message = i18n.global.t("KC_NUM_VALUE_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kc_related_time_is_blank
                error.error_message = i18n.global.t("KC_RELATED_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する。
            // 関連日時もこの画面で編集できるので比較に含める（含めないと日時だけ変えても保存できない）
            if (kc.title === title.value && kc.num_value === num_value.value &&
                moment(kc.related_time).toDate().getTime() === moment(related_date_string.value + " " + related_time_string.value).toDate().getTime()) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kc_is_no_update
                error.error_message = i18n.global.t("KC_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後KC情報を用意する
            const updated_kc = kc.clone()
            updated_kc.title = title.value
            updated_kc.num_value = num_value.value
            updated_kc.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            updated_kc.update_app = "gkill"
            updated_kc.update_device = props.application_config.device
            updated_kc.update_time = new Date(Date.now())
            updated_kc.update_user = props.application_config.user_id

            // 更新リクエストを飛ばす
            await delete_gkill_kyou_cache(updated_kc.id)
            const req = new UpdateKCRequest()
            req.want_response_kyou = true
            req.kc = updated_kc

            const res = await props.gkill_api.update_kc(req)
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
        title.value = cloned_kyou.value.typed_kc ? cloned_kyou.value.typed_kc.title : ""
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Init ──
    load()

    // ── Return ──
    return {
        // State
        is_loading,
        is_requested_submit,
        is_busy,
        cloned_kyou,
        title,
        num_value,
        related_date_typed,
        related_date_string,
        related_time_string,
        show_kyou,
        show_related_date_menu,
        show_related_time_menu,

        // Methods
        save,
        now_to_related_date_time,
        reset_related_date_time,
        reset,

        // Event relay
        crudRelayHandlers,
    }
}

