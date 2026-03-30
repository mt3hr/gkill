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
            @deleted_kyou="(deleted_kyou: Kyou) => emits('deleted_kyou', deleted_kyou)"
            @deleted_tag="(deleted_tag: Tag) => emits('deleted_tag', deleted_tag)"
            @deleted_text="(deleted_text: Text) => emits('deleted_text', deleted_text)"
            @deleted_notification="(deleted_notification: Notification) => emits('deleted_notification', deleted_notification)"
            @registered_kyou="(registered_kyou: Kyou) => emits('registered_kyou', registered_kyou)"
            @registered_tag="(registered_tag: Tag) => emits('registered_tag', registered_tag)"
            @registered_text="(registered_text: Text) => emits('registered_text', registered_text)"
            @registered_notification="(registered_notification: Notification) => emits('registered_notification', registered_notification)"
            @updated_kyou="(updated_kyou: Kyou) => emits('updated_kyou', updated_kyou)"
            @updated_tag="(updated_tag: Tag) => emits('updated_tag', updated_tag)"
            @updated_text="(updated_text: Text) => emits('updated_text', updated_text)"
            @updated_notification="(updated_notification: Notification) => emits('updated_notification', updated_notification)"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous: Kyou[], checked: boolean) => emits('requested_update_check_kyous', kyous, checked)"
            @requested_open_rykv_dialog="(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload)"
            ref="context_menu" />
    </div>
</template>
<script setup lang="ts">
import AttachedNotificationContextMenu from './attached-notification-context-menu.vue';
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"
import type { AttachedNotificationProps } from './attached-notification-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import type { Kyou } from '@/classes/datas/kyou';
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
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
