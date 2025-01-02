<template>
    <div>
        <div :class="notification_class" @contextmenu.prevent="async (e) => show_context_menu(e as PointerEvent)">
            <div class="notification_content">{{ notification.content }}</div>
            <div class="notification_time">{{ format_time(notification.notification_time) }}</div>
        </div>
        <AttachedNotificationContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :notification="notification" :kyou="kyou" :last_added_tag="last_added_tag"
            :highlight_targets="highlight_targets" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
            ref="context_menu" />
    </div>
</template>
<script setup lang="ts">
import AttachedNotificationContextMenu from './attached-notification-context-menu.vue';
import { computed, ref } from 'vue'
import type { AttachedNotificationProps } from './attached-notification-props';
import type { KyouViewEmits } from './kyou-view-emits';

const context_menu = ref<InstanceType<typeof AttachedNotificationContextMenu> | null>(null);

const props = defineProps<AttachedNotificationProps>()
const emits = defineEmits<KyouViewEmits>()

const notification_class = computed(() => {
    let highlighted = false;
    for (let i = 0; i < props.highlight_targets.length; i++) {
        if (props.highlight_targets[i].id === props.notification.id
            && props.highlight_targets[i].create_time.getTime() === props.notification.create_time.getTime()
            && props.highlight_targets[i].update_time.getTime() === props.notification.update_time.getTime()) {
            highlighted = true
            break
        }
    }
    if (highlighted) {
        return "highlighted_notification"
    }
    return "notification"
})


async function show_context_menu(e: PointerEvent): Promise<void> {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}

function format_time(time: Date) {
    let year: string | number = time.getFullYear()
    let month: string | number = time.getMonth() + 1
    let date: string | number = time.getDate()
    let hour: string | number = time.getHours()
    let minute: string | number = time.getMinutes()
    let second: string | number = time.getSeconds()
    const day_of_week = ['日', '月', '火', '水', '木', '金', '土'][time.getDay()]
    month = ('0' + month).slice(-2)
    date = ('0' + date).slice(-2)
    hour = ('0' + hour).slice(-2)
    minute = ('0' + minute).slice(-2)
    second = ('0' + second).slice(-2)
    return year + '/' + month + '/' + date + '(' + day_of_week + ')' + ' ' + hour + ':' + minute + ':' + second
}

</script>
<style lang="css" scoped>
.notification {
    background-color: #eee;
    border: dashed 1px;
    margin: 8px;
    padding: 8px;
}

.highlighted_notification {
    background-color: lightgreen;
    border: dashed 1px;
    margin: 8px;
    padding: 8px;
}

.notification_content {
    white-space: pre-line;
}
</style>