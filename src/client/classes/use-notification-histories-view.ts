import { type Ref, nextTick, ref, watch } from 'vue'
import { Notification } from '@/classes/datas/notification'
import type { NotificationHistoriesViewProps } from '@/pages/views/notification-histories-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useNotificationHistoriesView(options: {
    props: NotificationHistoriesViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    const cloned_notification: Ref<Notification> = ref(props.notification.clone())
    watch(() => props.notification, () => {
        cloned_notification.value = props.notification.clone()
        nextTick(() => cloned_notification.value.load_attached_histories())
    })
    nextTick(() => cloned_notification.value.load_attached_histories())

    return {
        cloned_notification,
    }
}
