<template>
    <NotificationView class="notification_history" v-for="notification in cloned_notification.attached_histories"
        :key="notification.id" :application_config="application_config" :gkill_api="gkill_api"
        :notification="notification" :kyou="kyou"
        :highlight_targets="highlight_targets" 
         :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" v-on="crudRelayHandlers" />
</template>
<script lang="ts" setup>
import NotificationView from './notification-view.vue';
import type { KyouViewEmits } from './kyou-view-emits'
import type { NotificationHistoriesViewProps } from './notification-histories-view-props';
import { useNotificationHistoriesView } from '@/classes/use-notification-histories-view'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

const props = defineProps<NotificationHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const crudRelayHandlers = build_kyou_view_relay(emits)

const {
    cloned_notification,
} = useNotificationHistoriesView({ props, emits })
</script>
<style lang="css">
.notification_history .highlighted_notification,
.notification_history .notification {
    width: 400px;
}
</style>
