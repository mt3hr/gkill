<template>
    <NotificationView class="notification_history" v-for="notification in cloned_notification.attached_histories"
        :key="notification.id" :application_config="application_config" :gkill_api="gkill_api"
        :notification="notification" :kyou="kyou" :last_added_tag="last_added_tag"
        :highlight_targets="highlight_targets" @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
        @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
        @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
</template>
<script lang="ts" setup>
import NotificationView from './notification-view.vue';
import { type Ref, nextTick, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { NotificationHistoriesViewProps } from './notification-histories-view-props';
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import type { Kyou } from '@/classes/datas/kyou';

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