import { type Ref, ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { UpdateTagRequest } from '@/classes/api/req_res/update-tag-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ConfirmDeleteTagViewProps } from '@/pages/views/confirm-delete-tag-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useConfirmDeleteTagView(options: {
    props: ConfirmDeleteTagViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(false)

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Delete logic ──
    //
    // 何があってもダイアログを閉じる。削除リクエストはサーバに届いているのに例外で
    // クローズまで到達せず、「消えているのに閉じない」状態になるのを防ぐため、
    // クローズはfinallyに置く（use-confirm-delete-kyou-view.ts と同じ形）。
    async function delete_tag(): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            // 更新後タグ情報を用意する
            const updated_tag = props.tag.clone()
            updated_tag.is_deleted = true
            updated_tag.update_app = "gkill"
            updated_tag.update_device = props.application_config.device
            updated_tag.update_time = new Date(Date.now())
            updated_tag.update_user = props.application_config.user_id

            // 更新リクエストを飛ばす
            await delete_gkill_kyou_cache(updated_tag.id)
            await delete_gkill_kyou_cache(updated_tag.target_id)
            const req = new UpdateTagRequest()
            req.tag = updated_tag
            const res = await props.gkill_api.update_tag(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits('deleted_tag', res.updated_tag)
            emits('requested_reload_kyou', props.kyou)
        } catch (err: unknown) {
            console.error(err)
            const error = new GkillError()
            error.error_code = GkillErrorCodes.failed_delete_tag
            error.error_message = i18n.global.t("FAILED_UPDATE_TAG_MESSAGE")
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

        // Methods
        delete_tag,

        // Event relay objects
        crudRelayHandlers,
    }
}

