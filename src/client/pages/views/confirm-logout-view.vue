<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("LOGOUT_TITLE") }}
        </v-card-title>
        <div>{{ i18n.global.t("CONFIRM_LOGOUT_MESSAGE") }}</div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="confirm_logout">{{ i18n.global.t("LOGOUT_TITLE") }}</v-btn>
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
import type { ConfirmLogoutViewEmits } from './confirm-logout-view-emits'
import type { ConfirmLogoutViewProps } from './confirm-logout-view-props'
import { useConfirmLogoutView } from '@/classes/use-confirm-logout-view'

const props = defineProps<ConfirmLogoutViewProps>()
const emits = defineEmits<ConfirmLogoutViewEmits>()

const {
    confirm_logout,
} = useConfirmLogoutView({ props, emits })
</script>
