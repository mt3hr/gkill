import { type Ref, ref } from 'vue'
import type { CreateAccountViewEmits } from '@/pages/views/create-account-view-emits'
import type { CreateAccountViewProps } from '@/pages/views/create-account-view-props'
import { AddAccountRequest } from '@/classes/api/req_res/add-account-request'

export function useCreateAccountView(options: {
    props: CreateAccountViewProps,
    emits: CreateAccountViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const new_user_id: Ref<string> = ref("")
    const do_not_initialize: Ref<boolean> = ref(false)

    // ── Business logic ──
    async function create_account(): Promise<void> {
        const req = new AddAccountRequest()
        req.account_info.is_enable = true
        req.account_info.is_admin = false
        req.account_info.password_reset_token = null
        req.account_info.user_id = new_user_id.value
        req.do_initialize = !do_not_initialize.value

        const res = await props.gkill_api.add_account(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }

        emits('created_account', res.added_account_info)
        emits('requested_reload_server_config')
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // State
        new_user_id,
        do_not_initialize,

        // Business logic
        create_account,
    }
}
