import { type Ref, ref, watch } from 'vue'
import type { ManageAccountViewProps } from '@/pages/views/manage-account-view-props'
import type { ManageAccountViewEmits } from '@/pages/views/manage-account-view-emits'
import type { Account } from '@/classes/datas/config/account'
import { UpdateAccountStatusRequest } from '@/classes/api/req_res/update-account-status-request'
import { GetServerConfigsRequest } from '@/classes/api/req_res/get-server-configs-request'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

export function useManageAccountView(options: {
    props: ManageAccountViewProps,
    emits: ManageAccountViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const allocate_rep_dialog = ref<any>(null)
    const confirm_reset_password_dialog = ref<any>(null)
    const create_account_dialog = ref<any>(null)
    const show_password_reset_link_dialog = ref<any>(null)

    // ── State refs ──
    const cloned_accounts: Ref<Array<Account>> = ref(props.server_configs[0].accounts)

    // ── Watchers ──
    watch(() => props.server_configs, () => {
        cloned_accounts.value = props.server_configs[0].accounts
    })

    // ── Business logic ──
    function show_create_account_dialog(): void {
        create_account_dialog.value?.show()
    }

    async function update_is_enable_account(account: Account, is_enable: boolean): Promise<void> {
        const req = new UpdateAccountStatusRequest()
        req.enable = is_enable
        req.target_user_id = account.user_id
        const res = await props.gkill_api.update_account_status(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }

        emits('requested_reload_server_config')
    }

    function show_allocate_rep_dialog(account: Account): void {
        allocate_rep_dialog.value?.show(account)
    }

    function show_confirm_reset_password_dialog(account: Account): void {
        confirm_reset_password_dialog.value?.show(account)
    }

    async function show_show_password_reset_link_dialog(account: Account): Promise<void> {
        const req = new GetServerConfigsRequest()
        const res = await props.gkill_api.get_server_configs(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        const accounts = res.server_configs.filter((server_config) => server_config.enable_this_device)[0].accounts
        for (let i = 0; i < accounts.length; i++) {
            if (account.user_id === accounts[i].user_id) {
                show_password_reset_link_dialog.value?.show(accounts[i])
            }
        }
    }

    // ── Event relay objects ──
    const allocateRepDialogHandlers = {
        'requested_reload_server_config': () => emits('requested_reload_server_config'),
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
    }

    const confirmResetPasswordDialogHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
        'requested_show_show_password_reset_dialog': (...account: any[]) => show_show_password_reset_link_dialog(account[0] as Account),
        'requested_reload_server_config': () => emits('requested_reload_server_config'),
    }

    const createAccountDialogHandlers = {
        'added_account': (...account: any[]) => show_show_password_reset_link_dialog(account[0] as Account),
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'requested_reload_server_config': () => emits('requested_reload_server_config'),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
    }

    const showPasswordResetLinkDialogHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
    }

    // ── Return ──
    return {
        // Template refs
        allocate_rep_dialog,
        confirm_reset_password_dialog,
        create_account_dialog,
        show_password_reset_link_dialog,

        // State
        cloned_accounts,

        // Business logic
        show_create_account_dialog,
        update_is_enable_account,
        show_allocate_rep_dialog,
        show_confirm_reset_password_dialog,
        show_show_password_reset_link_dialog,

        // Event relay objects
        allocateRepDialogHandlers,
        confirmResetPasswordDialogHandlers,
        createAccountDialogHandlers,
        showPasswordResetLinkDialogHandlers,
    }
}
