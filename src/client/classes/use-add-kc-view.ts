import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditKCViewProps } from '@/pages/views/edit-kc-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { KC } from '@/classes/datas/kc'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { AddKCRequest } from '@/classes/api/req_res/add-kc-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

export function useAddKcView(options: {
    props: EditKCViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)

    const kc: Ref<KC> = ref(((): KC => {
        const kc = new KC()
        kc.related_time = moment().toDate()
        return kc
    })())
    const title: Ref<string> = ref(kc.value.title)
    const num_value: Ref<number> = ref(kc.value.num_value)
    const related_date_typed: Ref<Date> = ref(moment(kc.value.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(kc.value.related_time).format("HH:mm:ss"))

    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Business logic ──
    function reset(): void {
        title.value = (kc.value.title)
        num_value.value = kc.value.num_value
        related_date_typed.value = (moment(kc.value.related_time).toDate())
        related_time_string.value = (moment(kc.value.related_time).format("HH:mm:ss"))
    }

    function reset_related_date_time(): void {
        related_date_typed.value = moment(kc.value.related_time).toDate()
        related_time_string.value = moment(kc.value.related_time).format("HH:mm:ss")
    }

    function now_to_related_date_time(): void {
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            if (!kc.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_kc_is_null
                error.error_message = i18n.global.t("CLIENT_KC_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kc_related_time_is_blank
                error.error_message = i18n.global.t("KC_RELATED_TIME_BLANK_MESSAGE")
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

            // 更新後KC情報を用意する
            const new_kc = new KC()
            new_kc.id = props.gkill_api.generate_uuid()
            new_kc.title = title.value
            new_kc.num_value = num_value.value
            new_kc.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            new_kc.create_app = "gkill"
            new_kc.create_device = props.application_config.device
            new_kc.create_time = new Date(Date.now())
            new_kc.create_user = props.application_config.user_id
            new_kc.update_app = "gkill"
            new_kc.update_device = props.application_config.device
            new_kc.update_time = new Date(Date.now())
            new_kc.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_kc.id)
            const req = new AddKCRequest()
            req.kc = new_kc
            req.want_response_kyou = true
            const res = await props.gkill_api.add_kc(req)
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

    function onUpdateRelatedDateMenu(value: boolean): void {
        show_related_date_menu.value = value
    }

    function onUpdateRelatedTimeMenu(value: boolean): void {
        show_related_time_menu.value = value
    }

    // ── Return ──
    return {
        // State
        is_requested_submit,
        kc,
        title,
        num_value,
        related_date_typed,
        related_date_string,
        related_time_string,
        show_related_date_menu,
        show_related_time_menu,

        // Business logic
        reset,
        reset_related_date_time,
        now_to_related_date_time,
        save,

        // Template event handlers
        onUpdateRelatedDateMenu,
        onUpdateRelatedTimeMenu,
    }
}
