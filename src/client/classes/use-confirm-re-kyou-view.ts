import { type Ref, ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { ReKyou } from '@/classes/datas/re-kyou'
import { AddReKyouRequest } from '@/classes/api/req_res/add-re-kyou-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ConfirmReKyouViewProps } from '@/pages/views/confirm-re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useConfirmReKyouView(options: {
    props: ConfirmReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(true)

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Rekyou logic ──
    //
    // idは呼ぶたびに新しく振るので、ガードが無いと連打したぶんだけリポストができる。
    // 何があってもダイアログを閉じるのは use-confirm-delete-kyou-view.ts と同じ理由
    async function rekyou(): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            // rekyou情報を用意する
            const new_rekyou = new ReKyou()
            new_rekyou.id = props.gkill_api.generate_uuid()
            new_rekyou.is_deleted = false
            new_rekyou.target_id = props.kyou.id
            new_rekyou.related_time = new Date(Date.now())
            new_rekyou.create_app = "gkill"
            new_rekyou.create_device = props.application_config.device
            new_rekyou.create_time = new Date(Date.now())
            new_rekyou.create_user = props.application_config.user_id
            new_rekyou.update_app = "gkill"
            new_rekyou.update_device = props.application_config.device
            new_rekyou.update_time = new Date(Date.now())
            new_rekyou.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_rekyou.id)
            const req = new AddReKyouRequest()
            req.want_response_kyou = true
            req.rekyou = new_rekyou
            const res = await props.gkill_api.add_rekyou(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            // 他のadd系と同じく、作ったものを一覧へ反映させる
            if (res.added_kyou) {
                emits('registered_kyou', res.added_kyou)
            }
            emits('requested_reload_list')
        } catch (err: unknown) {
            console.error(err)
            const error = new GkillError()
            error.error_code = GkillErrorCodes.failed_add_rekyou
            error.error_message = i18n.global.t("FAILED_ADD_REKYOU_MESSAGE")
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
        rekyou,

        // Event relay objects
        crudRelayHandlers,
    }
}

