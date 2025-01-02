<template>
    <v-card class="pa-2">
        <v-card-title>
            パスワードリセット
        </v-card-title>
        <div>下記アカウントのパスワードをリセットします</div>
        <div>処理完了後、パスワード再設定用リンクを表示します</div>
        <h1>{{ account.user_id }}</h1>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="reset_password">パスワードリセット</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="emits('requested_close_dialog')">キャンセル</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script setup lang="ts">
import { ResetPasswordRequest } from '@/classes/api/req_res/reset-password-request';
import type { ConfirmResetPasswordViewEmits } from './confirm-reset-password-view-emits'
import type { ConfirmResetPasswordViewProps } from './confirm-reset-password-view-props'

const props = defineProps<ConfirmResetPasswordViewProps>()
const emits = defineEmits<ConfirmResetPasswordViewEmits>()

async function reset_password(): Promise<void> {
    const req = new ResetPasswordRequest()
    req.session_id = props.gkill_api.get_session_id()
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
</script>
