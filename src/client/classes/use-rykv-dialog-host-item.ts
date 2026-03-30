import { computed, nextTick, onMounted, ref } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Notification } from '@/classes/datas/notification'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Kyou } from '@/classes/datas/kyou'
import type { OpenedRykvDialog } from '@/pages/views/rykv-dialog-kind'
import type { GkillPropsBase } from '@/pages/views/gkill-props-base'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import type { ComponentRef } from '@/classes/component-ref'

interface RykvDialogHostItemProps extends GkillPropsBase {
    item: OpenedRykvDialog
    enable_context_menu: boolean
    enable_dialog: boolean
}

interface RykvDialogHostItemEmits extends KyouDialogEmits {
    (e: 'closed', id: string): void
}

export function useRykvDialogHostItem(options: {
    props: RykvDialogHostItemProps,
    emits: RykvDialogHostItemEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const dialog = ref<ComponentRef | null>(null)

    // ── Computed ──
    const payload_tag = computed(() => (props.item.payload ?? null) as Tag | null)
    const payload_text = computed(() => (props.item.payload ?? null) as Text | null)
    const payload_notification = computed(() => (props.item.payload ?? null) as Notification | null)

    // ── Event relay objects ──
    const dialog_events = {
        closed: () => emits('closed', props.item.id),
        deleted_kyou: (kyou: Kyou) => emits('deleted_kyou', kyou),
        deleted_tag: (tag: Tag) => emits('deleted_tag', tag),
        deleted_text: (text: Text) => emits('deleted_text', text),
        deleted_notification: (notification: Notification) => emits('deleted_notification', notification),
        registered_kyou: (kyou: Kyou) => emits('registered_kyou', kyou),
        registered_tag: (tag: Tag) => emits('registered_tag', tag),
        registered_text: (text: Text) => emits('registered_text', text),
        registered_notification: (notification: Notification) => emits('registered_notification', notification),
        updated_kyou: (kyou: Kyou) => emits('updated_kyou', kyou),
        updated_tag: (tag: Tag) => emits('updated_tag', tag),
        updated_text: (text: Text) => emits('updated_text', text),
        updated_notification: (notification: Notification) => emits('updated_notification', notification),
        received_errors: (errors: Array<GkillError>) => emits('received_errors', errors),
        received_messages: (messages: Array<GkillMessage>) => emits('received_messages', messages),
        focused_kyou: (kyou: Kyou) => emits('focused_kyou', kyou),
        clicked_kyou: (kyou: Kyou) => {
            emits('focused_kyou', kyou)
            emits('clicked_kyou', kyou)
        },
        requested_reload_kyou: (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        requested_reload_list: () => emits('requested_reload_list'),
        requested_update_check_kyous: (kyous: Array<Kyou>, is_checked: boolean) =>
            emits('requested_update_check_kyous', kyous, is_checked),
        requested_open_rykv_dialog: (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) =>
            emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    // ── Lifecycle ──
    onMounted(async () => {
        await nextTick()
        dialog.value?.show?.()
    })

    // ── Return ──
    return {
        // Template refs
        dialog,

        // Computed
        payload_tag,
        payload_text,
        payload_notification,

        // Event relay objects
        dialog_events,
    }
}
