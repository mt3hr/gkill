<template>
    <div>
        <div :class="notification_class"
            @contextmenu.prevent="async (e: PointerEvent) => show_context_menu(e)">
            <div class="notification_content">{{ notification.content }}</div>
            <div class="notification_time">{{ format_time(notification.notification_time) }}</div>
        </div>
        <AttachedNotificationContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :notification="notification" :kyou="kyou"
            :highlight_targets="highlight_targets" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers"
            ref="context_menu" />
    </div>
</template>
<script setup lang="ts">
import AttachedNotificationContextMenu from './attached-notification-context-menu.vue';
import type { AttachedNotificationProps } from './attached-notification-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { useAttachedNotification } from '@/classes/use-attached-notification';

const props = defineProps<AttachedNotificationProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // State
    context_menu,
    notification_class,

    // Business logic
    show_context_menu,
    format_time,
    // Event relay objects
    crudRelayHandlers,
} = useAttachedNotification({ props, emits })
</script>
<style lang="css" scoped>
.notification {
    background-color: var(--v-attached-text-background-base);
    border: solid 1px;
    margin: 8px;
    padding: 8px;
}

.highlighted_notification {
    background-color: rgb(var(--v-theme-highlight));
    border: solid 1px;
    margin: 8px;
    padding: 8px;
}

.notification_content {
    white-space: pre-line;
}
</style>
