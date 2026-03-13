import type { NotificationViewProps } from '@/pages/views/notification-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useNotificationView(options: {
    props: NotificationViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    return {
    }
}
