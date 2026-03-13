import { computed, ref } from 'vue'
import moment from 'moment'
import { format_duration } from '@/classes/format-date-time'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { TimeIsViewProps } from '@/pages/views/time-is-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type TimeIsContextMenu from '@/pages/views/time-is-context-menu.vue'
import type EndTimeIsPlaingDialog from '@/pages/dialogs/end-time-is-plaing-dialog.vue'

export function useTimeIsView(options: {
    props: TimeIsViewProps,
    emits: KyouViewEmits,
    context_menu: ReturnType<typeof ref<InstanceType<typeof TimeIsContextMenu> | null>>,
    end_timeis_plaing_dialog: ReturnType<typeof ref<InstanceType<typeof EndTimeIsPlaingDialog> | null>>,
}) {
    const { props, emits, context_menu, end_timeis_plaing_dialog } = options

    // ── Computed ──
    const duration = computed(() => {
        let time1 = props.timeis.start_time
        let time2 = props.timeis.end_time

        time2 = time2 ? time2 : moment().toDate()
        const diff = Math.abs(time2.getTime() - time1.getTime())
        return format_duration(diff).replace("<br>", " ")
    })

    // ── CRUD relay handlers ──
    const crudRelayHandlers = {
        'deleted_kyou': (...args: any[]) => emits('deleted_kyou', args[0] as Kyou),
        'deleted_tag': (...args: any[]) => emits('deleted_tag', args[0] as Tag),
        'deleted_text': (...args: any[]) => emits('deleted_text', args[0] as Text),
        'deleted_notification': (...args: any[]) => emits('deleted_notification', args[0] as Notification),
        'registered_kyou': (...args: any[]) => emits('registered_kyou', args[0] as Kyou),
        'registered_tag': (...args: any[]) => emits('registered_tag', args[0] as Tag),
        'registered_text': (...args: any[]) => emits('registered_text', args[0] as Text),
        'registered_notification': (...args: any[]) => emits('registered_notification', args[0] as Notification),
        'updated_kyou': (...args: any[]) => emits('updated_kyou', args[0] as Kyou),
        'updated_tag': (...args: any[]) => emits('updated_tag', args[0] as Tag),
        'updated_text': (...args: any[]) => emits('updated_text', args[0] as Text),
        'updated_notification': (...args: any[]) => emits('updated_notification', args[0] as Notification),
        'received_errors': (...args: any[]) => emits('received_errors', args[0] as Array<GkillError>),
        'received_messages': (...args: any[]) => emits('received_messages', args[0] as Array<GkillMessage>),
        'requested_reload_kyou': (...args: any[]) => emits('requested_reload_kyou', args[0] as Kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (...args: any[]) => emits('requested_update_check_kyous', args[0] as Array<Kyou>, args[1] as boolean),
        'requested_open_rykv_dialog': (...args: any[]) => emits('requested_open_rykv_dialog', args[0], args[1], args[2]),
    }

    // ── Methods ──
    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    function show_end_timeis_dialog(): void {
        end_timeis_plaing_dialog.value?.show()
    }

    // ── Return ──
    return {
        // State
        duration,

        // Methods
        show_context_menu,
        show_end_timeis_dialog,

        // Event relay objects
        crudRelayHandlers,
    }
}
