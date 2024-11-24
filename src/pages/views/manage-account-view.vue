<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>アカウント管理</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn color="primary" @click="show_create_account_dialog">アカウント追加</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <table>
                <tr v-for="account in cloned_accounts" :key="account.user_id">
                    <td>
                        <v-checkbox v-model="account.is_enable"
                            @click="update_is_enable_account(account, !account.is_enable)" />
                    </td>
                    <td>
                        {{ account.user_id }}
                    </td>
                    <td>
                        <v-btn color="primary" @click="show_allocate_rep_dialog(account)">Rep割当管理</v-btn>
                    </td>
                    <td>
                        <v-btn v-if="!account.password_reset_token" color="primary"
                            @click="show_confirm_reset_password_dialog(account)">パスワードリセット</v-btn>
                        <v-btn v-if="account.password_reset_token" color="primary"
                            @click="show_show_password_reset_link_dialog(account)">パスワードリセット中</v-btn>
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">閉じる</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AllocateRepDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config"
            @requested_reload_server_config="(server_config) => emits('requested_reload_server_config', server_config)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="allocate_rep_dialog" />
        <ConfirmResetPasswordDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_show_show_password_reset_dialog="(account) => show_show_password_reset_link_dialog(account)"
            @requested_reload_server_config="(server_config) => emits('requested_reload_server_config', server_config)"
            ref="confirm_reset_password_dialog" />
        <CreateAccountDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_reload_server_config="(server_config) => emits('requested_reload_server_config', server_config)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="create_account_dialog" />
        <ShowPasswordResetLinkDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            ref="show_password_reset_link_dialog" />
    </v-card>
</template>
<script setup lang="ts">
import { type Ref, ref, watch } from 'vue'
import type { ManageAccountViewEmits } from './manage-account-view-emits'
import type { ManageAccountViewProps } from './manage-account-view-props'
import AllocateRepDialog from '../dialogs/allocate-rep-dialog.vue'
import ConfirmResetPasswordDialog from '../dialogs/confirm-reset-password-dialog.vue'
import CreateAccountDialog from '../dialogs/create-account-dialog.vue'
import ShowPasswordResetLinkDialog from '../dialogs/show-password-reset-link-dialog.vue'
import type { Account } from '@/classes/datas/config/account'
import { GkillAPI } from '@/classes/api/gkill-api'
import { UpdateAccountStatusRequest } from '@/classes/api/req_res/update-account-status-request'
import { GetServerConfigRequest } from '@/classes/api/req_res/get-server-config-request'

const allocate_rep_dialog = ref<InstanceType<typeof AllocateRepDialog> | null>(null);
const confirm_reset_password_dialog = ref<InstanceType<typeof ConfirmResetPasswordDialog> | null>(null);
const create_account_dialog = ref<InstanceType<typeof CreateAccountDialog> | null>(null);
const show_password_reset_link_dialog = ref<InstanceType<typeof ShowPasswordResetLinkDialog> | null>(null);

const props = defineProps<ManageAccountViewProps>()
const emits = defineEmits<ManageAccountViewEmits>()

const cloned_accounts: Ref<Array<Account>> = ref(props.server_config.accounts)

watch(() => props.server_config, () => {
    cloned_accounts.value = props.server_config.accounts
})

function show_create_account_dialog(): void {
    create_account_dialog.value?.show()
}

async function update_is_enable_account(account: Account, is_enable: boolean): Promise<void> {
    const req = new UpdateAccountStatusRequest()
    req.enable = is_enable
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.target_user_id = account.user_id
    const res = await GkillAPI.get_instance().update_account_status(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const server_config_req = new GetServerConfigRequest()
    server_config_req.session_id = GkillAPI.get_instance().get_session_id()
    const server_config_res = await GkillAPI.get_instance().get_server_config(server_config_req)
    if (server_config_res.errors && server_config_res.errors.length !== 0) {
        emits('received_errors', server_config_res.errors)
        return
    }
    if (server_config_res.messages && server_config_res.messages.length !== 0) {
        emits('received_messages', server_config_res.messages)
    }

    emits('requested_reload_server_config', server_config_res.server_config)
}

function show_allocate_rep_dialog(account: Account): void {
    allocate_rep_dialog.value?.show(account)
}

function show_confirm_reset_password_dialog(account: Account): void {
    confirm_reset_password_dialog.value?.show(account)
}

function show_show_password_reset_link_dialog(account: Account): void {
    show_password_reset_link_dialog.value?.show(account)
}
</script>
