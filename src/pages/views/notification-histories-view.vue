<template>
    <NotificationView class="notification_history" v-for="notification in cloned_notification.attached_histories"
        :key="notification.id" :application_config="application_config" :gkill_api="gkill_api"
        :notification="notification" :kyou="kyou" :last_added_tag="last_added_tag"
        :highlight_targets="highlight_targets" @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
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
        @received_errors="(errors) => emits('received_errors', errors)" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import NotificationView from './notification-view.vue';
import { type Ref, nextTick, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { Notification } from '@/classes/datas/notification';
import type { NotificationHistoriesViewProps } from './notification-histories-view-props';

const props = defineProps<NotificationHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_notification: Ref<Notification> = ref(props.notification.clone())
watch(() => props.notification, () => {
    cloned_notification.value = props.notification.clone()
    nextTick(() => cloned_notification.value.load_attached_histories())
})
nextTick(() => cloned_notification.value.load_attached_histories())

</script>
<style lang="css">
.notification_history .highlighted_notification,
.notification_history .notification {
    width: 400px;
}
</style>