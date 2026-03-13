<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("MANAGE_ACCOUNT_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_create_account_dialog">{{
                        i18n.global.t("ADD_ACCOUNT_TITLE")
                    }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card class="manage_account_list_view_card">
            <table class="manage_account_list_view">
                <tr v-for="account in cloned_accounts" :key="account.user_id">
                    <td>
                        <v-checkbox v-model="account.is_enable"
                            @click="update_is_enable_account(account, !account.is_enable)" />
                    </td>
                    <td>
                        {{ account.user_id }}
                    </td>
                    <td>
                        <v-btn dark color="primary" @click="show_allocate_rep_dialog(account)">{{
                            i18n.global.t("ALLOCATE_REP_TITLE") }}</v-btn>
                    </td>
                    <td>
                        <v-btn dark color="primary" v-if="!account.password_reset_token"
                            @click="show_confirm_reset_password_dialog(account)">{{
                                i18n.global.t("RESETED_PASSWORD_TITLE")
                            }}</v-btn>
                        <v-btn dark color="primary" v-if="account.password_reset_token"
                            @click="show_show_password_reset_link_dialog(account)">{{
                                i18n.global.t("RESETTING_PASSWORD_TITLE")
                            }}</v-btn>
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{
                        i18n.global.t("CLOSE_TITLE")
                    }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AllocateRepDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs"
            v-on="allocateRepDialogHandlers"
            ref="allocate_rep_dialog" />
        <ConfirmResetPasswordDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs"
            v-on="confirmResetPasswordDialogHandlers"
            ref="confirm_reset_password_dialog" />
        <CreateAccountDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs"
            v-on="createAccountDialogHandlers"
            ref="create_account_dialog" />
        <ShowPasswordResetLinkDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs"
            v-on="showPasswordResetLinkDialogHandlers"
            ref="show_password_reset_link_dialog" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { ManageAccountViewEmits } from './manage-account-view-emits'
import type { ManageAccountViewProps } from './manage-account-view-props'
import AllocateRepDialog from '../dialogs/allocate-rep-dialog.vue'
import ConfirmResetPasswordDialog from '../dialogs/confirm-reset-password-dialog.vue'
import CreateAccountDialog from '../dialogs/create-account-dialog.vue'
import ShowPasswordResetLinkDialog from '../dialogs/show-password-reset-link-dialog.vue'
import { useManageAccountView } from '@/classes/use-manage-account-view'

const props = defineProps<ManageAccountViewProps>()
const emits = defineEmits<ManageAccountViewEmits>()

const {
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
} = useManageAccountView({ props, emits })
</script>
<style lang="css" scoped>
.manage_account_list_view_card {
    overflow-x: scroll;
}

.manage_account_list_view {
    width: max-content;
}
</style>
