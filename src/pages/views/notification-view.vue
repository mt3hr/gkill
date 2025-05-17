<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <AttachedNotification :notification="notification" :application_config="application_config"
                :gkill_api="gkill_api" :kyou="kyou" :last_added_tag="last_added_tag"
                :highlight_targets="highlight_targets" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
                @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
                @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
                @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
                @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
                @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
                @registered_text="(registered_text) => emits('registered_text', registered_text)"
                @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
                @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
                @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
                @updated_text="(updated_text) => emits('updated_text', updated_text)"
                @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pa-0 ma-0">
            <span class="update_time">
                {{ format_time(notification.update_time) }}
            </span>
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pa-0 ma-0">
            <span class="update_device">
                {{ notification.update_device }}
            </span>
        </v-col>
    </v-row>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import AttachedNotification from './attached-notification.vue';
import type { KyouViewEmits } from './kyou-view-emits'
import moment from 'moment';
import type { NotificationViewProps } from './notification-view-props';

defineProps<NotificationViewProps>()
const emits = defineEmits<KyouViewEmits>()

function format_time(time: Date): string {
    return moment(time).format("yyyy-MM-DD HH:mm:ss")
}
</script>
