<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>パスワードリセット</span>
                </v-col>
            </v-row>
        </v-card-title>
        <div>下記アカウントのパスワードリセットが完了しました。</div>
        <div>記載URLをユーザに連絡し、パスワードリセットを実施してください</div>
        <div>{{ account.user_id }}</div>

        <v-text-field v-model="lan_password_reset_url" label="LAN内" readonly @click="copy_lan_password_reset_url"
            @focus="$event.target.select()" />
        <v-text-field v-model="over_lan_password_reset_url" label="LAN外" readonly
            @click="copy_over_lan_password_reset_url" @focus="$event.target.select()" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">閉じる</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue';
import type { ShowPasswordResetLinkViewEmits } from './show-password-reset-link-view-emits'
import type { ShowPasswordResetLinkViewProps } from './show-password-reset-link-view-props'
import { GkillMessage } from '@/classes/api/gkill-message';

const props = defineProps<ShowPasswordResetLinkViewProps>()
const emits = defineEmits<ShowPasswordResetLinkViewEmits>()

const lan_password_reset_url: Ref<string> = ref("")
const over_lan_password_reset_url: Ref<string> = ref("")

watch(() => props.account, () => update_password_reset_urls())

update_password_reset_urls()

function update_password_reset_urls(): void {
    const current_server_config = props.server_configs.filter((server_config) => server_config.enable_this_device)[0]
    const token = props.account.password_reset_token
    let http = current_server_config.enable_tls ? "https://" : "http://"
    const port = current_server_config.address
    lan_password_reset_url.value = `${http}localhost${port}/set_new_password?reset_token=${token}`
    over_lan_password_reset_url.value = `${http}localhost${port}/set_new_password?reset_token=${token}`
}
function copy_lan_password_reset_url(): void {
    navigator.clipboard.writeText(lan_password_reset_url.value);
    const message = new GkillMessage()
    message.message_code = "//TODO"
    message.message = "コピーしました"
    emits('received_messages', [message])
}
function copy_over_lan_password_reset_url(): void {
    navigator.clipboard.writeText(over_lan_password_reset_url.value);
    const message = new GkillMessage()
    message.message_code = "//TODO"
    message.message = "コピーしました"
    emits('received_messages', [message])
}
</script>
