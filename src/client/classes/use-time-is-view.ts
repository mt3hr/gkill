import { computed, ref } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
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
        const time1 = props.timeis.start_time
        let time2 = props.timeis.end_time

        time2 = time2 ? time2 : moment().toDate()
        const diff = Math.abs(time2.getTime() - time1.getTime())
        return format_duration(diff).replace("<br>", " ")
    })

    // ── CRUD relay handlers ──
    const crudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => emits('deleted_kyou', kyou),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => emits('registered_kyou', kyou),
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => emits('updated_kyou', kyou),
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_reload_kyou': (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => emits('requested_update_check_kyous', kyous, checked),
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
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
