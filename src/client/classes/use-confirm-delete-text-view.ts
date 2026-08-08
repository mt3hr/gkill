import { computed, type Ref, ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import { UpdateTextRequest } from '@/classes/api/req_res/update-text-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ConfirmDeleteTextViewProps } from '@/pages/views/confirm-delete-text-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useConfirmDeleteTextView(options: {
    props: ConfirmDeleteTextViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(true)

    const text_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifier = props.text.generate_info_identifier()
        return [info_identifier]
    })

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Delete logic ──
    //
    // 何があってもダイアログを閉じる。理由は use-confirm-delete-kyou-view.ts と同じ
    async function delete_text(): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            // 更新後テキスト情報を用意する
            const updated_text = props.text.clone()
            updated_text.is_deleted = true
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
            emits('deleted_text', res.updated_text)
            emits('requested_reload_kyou', props.kyou)
        } catch (err: unknown) {
            console.error(err)
            const error = new GkillError()
            error.error_code = GkillErrorCodes.failed_delete_text
            error.error_message = i18n.global.t("FAILED_UPDATE_TEXT_MESSAGE")
            emits('received_errors', [error])
        } finally {
            is_requested_submit.value = false
            emits('requested_close_dialog')
        }
    }

    // ── Return ──
    return {
        // State
        is_requested_submit,
        show_kyou,
        text_highlight_targets,

        // Methods
        delete_text,

        // Event relay objects
        crudRelayHandlers,
    }
}

