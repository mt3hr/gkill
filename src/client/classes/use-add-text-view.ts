import { ref, type Ref } from 'vue'
import { i18n } from '@/i18n'
import { Text } from '@/classes/datas/text'
import { AddTextRequest } from '@/classes/api/req_res/add-text-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { AddTextViewProps } from '@/pages/views/add-text-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useAddTextView(options: {
    props: AddTextViewProps,
    emits: KyouViewEmits
}) {
    const props = options.props
    const emits = options.emits

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(false)
    const text_value: Ref<string> = ref("")

    // ── Methods ──
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

            // テキスト情報を用意する
            const new_text = new Text()
            new_text.text = text_value.value
            new_text.id = props.gkill_api.generate_uuid()
            new_text.is_deleted = false
            new_text.target_id = props.kyou.id
            new_text.related_time = new Date(Date.now())
            new_text.create_app = "gkill"
            new_text.create_device = props.application_config.device
            new_text.create_time = new Date(Date.now())
            new_text.create_user = props.application_config.user_id
            new_text.update_app = "gkill"
            new_text.update_app = "gkill"
            new_text.update_device = props.application_config.device
            new_text.update_time = new Date(Date.now())
            new_text.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_text.id)
            await delete_gkill_kyou_cache(new_text.target_id)
            const req = new AddTextRequest()
            req.text = new_text
            const res = await props.gkill_api.add_text(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    return {
        // State
        is_requested_submit,
        show_kyou,
        text_value,

        // Methods
        save,
    }
}
