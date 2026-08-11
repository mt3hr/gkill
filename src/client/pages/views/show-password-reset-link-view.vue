<template>
    <v-card variant="flat">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("RESETED_PASSWORD_TITLE") }}</span>
                </v-col>
            </v-row>
        </v-card-title>
        <div>
            <pre>{{ i18n.global.t("RESETED_PASSWORD_MESSAGE") }}</pre>
        </div>
        <div>{{ account.user_id }}</div>
        <div v-if="password_reset_token_expiration_label !== ''" class="password_reset_token_expiration">
            {{ i18n.global.t("PASSWORD_RESET_TOKEN_EXPIRATION_TITLE") }}: {{ password_reset_token_expiration_label }}
        </div>
        <v-alert v-if="is_password_reset_link_expired" type="warning" variant="tonal" density="compact" class="my-2">
            {{ i18n.global.t("PASSWORD_RESET_LINK_EXPIRED_MESSAGE") }}
        </v-alert>

        <v-text-field v-model="local_password_reset_url" :label="i18n.global.t('LOCAL_TITLE')" readonly
            @click="copy_local_password_reset_url" @focus="$event.target.select()" />
        <v-text-field v-model="lan_password_reset_url" :label="i18n.global.t('IN_LAN_TITLE')" readonly
            @click="copy_lan_password_reset_url" @focus="$event.target.select()" />
        <v-text-field v-model="over_lan_password_reset_url" :label="i18n.global.t('OVER_LAN_TITLE')" readonly
            @click="copy_over_lan_password_reset_url" @focus="$event.target.select()" />
        <v-card-action>
            <v-row class="pa-0 ma-0 gkill-dialog-actions">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" :loading="is_reissuing" :disabled="is_reissuing"
                        @click="reissue_password_reset_link()">{{
                            i18n.global.t("REISSUE_PASSWORD_RESET_LINK_TITLE")
                        }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ i18n.global.t("CLOSE_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ShowPasswordResetLinkViewEmits } from './show-password-reset-link-view-emits'
import type { ShowPasswordResetLinkViewProps } from './show-password-reset-link-view-props'
import { useShowPasswordResetLinkView } from '@/classes/use-show-password-reset-link-view'

const props = defineProps<ShowPasswordResetLinkViewProps>()
const emits = defineEmits<ShowPasswordResetLinkViewEmits>()

const {
    local_password_reset_url,
    lan_password_reset_url,
    over_lan_password_reset_url,
    is_reissuing,
    password_reset_token_expiration_label,
    is_password_reset_link_expired,
    copy_local_password_reset_url,
    copy_lan_password_reset_url,
    copy_over_lan_password_reset_url,
    reissue_password_reset_link,
} = useShowPasswordResetLinkView({ props, emits })
</script>
<style lang="css" scoped>
.password_reset_token_expiration {
    font-size: small;
    opacity: 0.8;
}
</style>
