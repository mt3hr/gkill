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
            :server_configs="server_configs"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="allocate_rep_dialog" />
        <ConfirmResetPasswordDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_show_show_password_reset_dialog="(account) => show_show_password_reset_link_dialog(account)"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            ref="confirm_reset_password_dialog" />
        <CreateAccountDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs" @added_account="(account) => show_show_password_reset_link_dialog(account)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @received_messages="(messages) => emits('received_messages', messages)" ref="create_account_dialog" />
        <ShowPasswordResetLinkDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs" @received_errors="(errors) => emits('received_errors', errors)"
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
import { UpdateAccountStatusRequest } from '@/classes/api/req_res/update-account-status-request'
import { GetServerConfigsRequest } from '@/classes/api/req_res/get-server-configs-request'

const allocate_rep_dialog = ref<InstanceType<typeof AllocateRepDialog> | null>(null);
const confirm_reset_password_dialog = ref<InstanceType<typeof ConfirmResetPasswordDialog> | null>(null);
const create_account_dialog = ref<InstanceType<typeof CreateAccountDialog> | null>(null);
const show_password_reset_link_dialog = ref<InstanceType<typeof ShowPasswordResetLinkDialog> | null>(null);

const props = defineProps<ManageAccountViewProps>()
const emits = defineEmits<ManageAccountViewEmits>()

const cloned_accounts: Ref<Array<Account>> = ref(props.server_configs[0].accounts)

watch(() => props.server_configs, () => {
    cloned_accounts.value = props.server_configs[0].accounts
})

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
</script>
