<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("RESET_PASSWORD_TITLE") }}
        </v-card-title>
        <div>{{ i18n.global.t("RESET_PASSWORD_MESSAGE") }}</div>
        <h1>{{ account.user_id }}</h1>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="reset_password">{{ i18n.global.t("RESET_PASSWORD_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ i18n.global.t("CANCEL_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { ConfirmResetPasswordViewEmits } from './confirm-reset-password-view-emits'
import type { ConfirmResetPasswordViewProps } from './confirm-reset-password-view-props'
import { useConfirmResetPasswordView } from '@/classes/use-confirm-reset-password-view'

const props = defineProps<ConfirmResetPasswordViewProps>()
const emits = defineEmits<ConfirmResetPasswordViewEmits>()

const {
    reset_password,
} = useConfirmResetPasswordView({ props, emits })
</script>
