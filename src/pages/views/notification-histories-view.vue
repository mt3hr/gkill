<template>
    <NotificationView v-for="notification in cloned_notification.attached_histories" :key="notification.id"
        :application_config="application_config" :gkill_api="gkill_api" :notification="notification" :kyou="kyou"
        :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
        @received_errors="(errors) => emits('received_errors', errors)" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
</template>
<script lang="ts" setup>
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
