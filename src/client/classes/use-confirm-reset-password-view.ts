import { ResetPasswordRequest } from '@/classes/api/req_res/reset-password-request'
import type { ConfirmResetPasswordViewEmits } from '@/pages/views/confirm-reset-password-view-emits'
import type { ConfirmResetPasswordViewProps } from '@/pages/views/confirm-reset-password-view-props'

export function useConfirmResetPasswordView(options: {
    props: ConfirmResetPasswordViewProps,
    emits: ConfirmResetPasswordViewEmits,
}) {
    const { props, emits } = options

    // ── Methods ──
    async function reset_password(): Promise<void> {
        const req = new ResetPasswordRequest()
        req.target_user_id = props.account.user_id
        const res = await props.gkill_api.reset_password(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }

        emits('requested_reload_server_config')
        emits('requested_show_show_password_reset_dialog', props.account)
    }

    // ── Return ──
    return {
        // Methods
        reset_password,
    }
}
