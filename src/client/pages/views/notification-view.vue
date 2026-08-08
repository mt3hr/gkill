<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <AttachedNotification :notification="notification" :application_config="application_config"
                :gkill_api="gkill_api" :kyou="kyou"
                :highlight_targets="highlight_targets" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" v-on="crudRelayHandlers" />
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
import { format_time } from '@/classes/format-date-time'
import type { NotificationViewProps } from './notification-view-props';
import { useNotificationView } from '@/classes/use-notification-view'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

const props = defineProps<NotificationViewProps>()
const emits = defineEmits<KyouViewEmits>()

const crudRelayHandlers = build_kyou_view_relay(emits)

const {
} = useNotificationView({ props, emits })
</script>
