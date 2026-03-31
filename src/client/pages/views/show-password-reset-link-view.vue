<template>
    <v-card>
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

        <v-text-field v-model="local_password_reset_url" :label="i18n.global.t('LOCAL_TITLE')" readonly
            @click="copy_local_password_reset_url" @focus="$event.target.select()" />
        <v-text-field v-model="lan_password_reset_url" :label="i18n.global.t('IN_LAN_TITLE')" readonly
            @click="copy_lan_password_reset_url" @focus="$event.target.select()" />
        <v-text-field v-model="over_lan_password_reset_url" :label="i18n.global.t('OVER_LAN_TITLE')" readonly
            @click="copy_over_lan_password_reset_url" @focus="$event.target.select()" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
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
    copy_local_password_reset_url,
    copy_lan_password_reset_url,
    copy_over_lan_password_reset_url,
} = useShowPasswordResetLinkView({ props, emits })
</script>
