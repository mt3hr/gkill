'use strict'

import { computed, ref } from 'vue'
import { i18n } from '@/i18n'
import type { AttachedNotificationProps } from '@/pages/views/attached-notification-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import AttachedNotificationContextMenu from '@/pages/views/attached-notification-context-menu.vue'

export function useAttachedNotification(options: {
    props: AttachedNotificationProps
    emits: KyouViewEmits
}) {
    const { props, emits: _emits } = options

    const context_menu = ref<InstanceType<typeof AttachedNotificationContextMenu> | null>(null)

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
        const year: string | number = time.getFullYear()
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

    return {
        context_menu,
        notification_class,
        show_context_menu,
        format_time,
    }
}
