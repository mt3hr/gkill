import { computed, type Ref, ref } from 'vue'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { TagHistoriesDialogProps } from '@/pages/dialogs/tag-histories-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useTagHistoriesDialog(options: {
    props: TagHistoriesDialogProps,
    emits: KyouDialogEmits,
}) {
    const { props, emits } = options

    // ── Dialog state ──
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("tag-histories-dialog", {
        centerMode: "always",
    })

    const show_kyou: Ref<boolean> = ref(false)

    // ── Computed ──
    const tag_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifer = props.tag.generate_info_identifer()
        return [info_identifer]
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
        'focused_kyou': (...args: any[]) => emits('focused_kyou', args[0] as Kyou),
        'clicked_kyou': (...args: any[]) => { emits('focused_kyou', args[0] as Kyou); emits('clicked_kyou', args[0] as Kyou) },
        'requested_reload_kyou': (...args: any[]) => emits('requested_reload_kyou', args[0] as Kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (...args: any[]) => emits('requested_update_check_kyous', args[0] as Array<Kyou>, args[1] as boolean),
    }

    // ── Methods ──
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
        emits('closed')
    }

    // ── Return ──
    return {
        // State
        is_show_dialog,
        ui,
        show_kyou,
        tag_highlight_targets,

        // Methods
        show,
        hide,

        // Event relay objects
        crudRelayHandlers,
    }
}
