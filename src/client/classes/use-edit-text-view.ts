import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditTextViewProps } from '@/pages/views/edit-text-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { UpdateTextRequest } from '@/classes/api/req_res/update-text-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { Kyou } from '@/classes/datas/kyou'
import type { Text } from '@/classes/datas/text'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useEditTextView(options: {
    props: EditTextViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_loading = ref(true)
    const is_requested_submit = ref(false)
    const is_busy = computed(() => is_loading.value || is_requested_submit.value)
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const cloned_text: Ref<Text> = ref(props.text.clone())
    const text_value: Ref<string> = ref(cloned_text.value.text)
    const show_kyou: Ref<boolean> = ref(false)

    // ── Business logic ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.load_typed_datas()
            cloned_text.value = props.text.clone()
            text_value.value = cloned_text.value.text
        } finally {
            is_loading.value = false
        }
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // 値がなかったらエラーメッセージを出力する
            if (text_value.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.text_is_blank
                error.error_message = i18n.global.t("TEXT_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する
            if (cloned_text.value.text === text_value.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.text_is_no_update
                error.error_message = i18n.global.t("TEXT_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後テキスト情報を用意する
            const updated_text = cloned_text.value.clone()
            updated_text.text = text_value.value
            updated_text.update_app = "gkill"
            updated_text.update_device = props.application_config.device
            updated_text.update_time = new Date(Date.now())
            updated_text.update_user = props.application_config.user_id

            // 更新リクエストを飛ばす
            await delete_gkill_kyou_cache(updated_text.id)
            await delete_gkill_kyou_cache(updated_text.target_id)
            const req = new UpdateTextRequest()
            req.text = updated_text
            const res = await props.gkill_api.update_text(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits("updated_text", res.updated_text)
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Init calls ──
    load()

    // ── Return ──
    return {
        // State
        is_loading,
        is_requested_submit,
        is_busy,
        cloned_kyou,
        cloned_text,
        text_value,
        show_kyou,

        // Business logic
        save,

        // Event relay objects
        crudRelayHandlers,
    }
}

