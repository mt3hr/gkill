<template>
    <div>
        <div :class="notification_class" @contextmenu.prevent="async (e) => show_context_menu(e as PointerEvent)">
            <div class="notification_content">{{ notification.content }}</div>
            <div class="notification_time">{{ format_time(notification.notification_time) }}</div>
        </div>
        <AttachedNotificationContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :notification="notification" :kyou="kyou" :last_added_tag="last_added_tag"
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
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
            ref="context_menu" />
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
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
    const day_of_week = [i18n.global.t("SUNDAY_TITLE"), i18n.global.t("MONDAY_TITLE"), i18n.global.t("TUESDAY_TITLE"), i18n.global.t("WEDNESDAY_TITLE"), i18n.global.t("THURSDAY_TITLE"), i18n.global.t("FRIDAY_TITLE"), i18n.global.t("SATURDAY_TITLE")][time.getDay()]
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