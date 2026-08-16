import { type Ref, ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { remove_attached_tags } from '@/classes/kyou-tags'
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
            // 削除は is_deleted=true の版を足す。手順は classes/kyou-tags.ts に集約してある
            const result = await remove_attached_tags(props.gkill_api, props.application_config, [props.tag])
            if (result.errors.length !== 0) {
                emits('received_errors', result.errors)
                return
            }
            if (result.messages.length !== 0) {
                emits('received_messages', result.messages)
            }
            result.removed_tags.forEach(removed_tag => emits('deleted_tag', removed_tag))
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

