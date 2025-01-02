<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <AttachedNotification :notification="notification" :application_config="application_config"
                :gkill_api="gkill_api" :kyou="kyou" :last_added_tag="last_added_tag"
                :highlight_targets="highlight_targets" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
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
