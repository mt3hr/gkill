import { computed, type Ref, ref } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { DecideRelatedTimeUploadedFileViewProps } from '@/pages/views/decide-related-time-uploaded-file-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { ComponentRef } from '@/classes/component-ref'

export function useDecideRelatedTimeUploadedFileView(options: {
    props: DecideRelatedTimeUploadedFileViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_list_view = ref<ComponentRef | null>(null)
    const edit_idf_kyou_view = ref<ComponentRef | null>(null)

    // ── State refs ──
    const enable_context_menu = ref(true)
    const enable_dialog = ref(true)
    const focused_kyou: Ref<Kyou | null> = ref(null)
    const kyou_height: Ref<number> = ref(180)
    const kyou_height_px = computed(() => kyou_height.value ? kyou_height.value.toString().concat("px") : "0px")

    // ── Business logic ──
    async function reload_focused_kyou(): Promise<void> {
        if (!focused_kyou.value) {
            return
        }
        const updated_kyou = focused_kyou.value.clone()
        await updated_kyou.reload(false, true)
        await updated_kyou.load_all()
        focused_kyou.value = updated_kyou
    }

    function removeKyouFromListById(list: Array<Kyou>, deletedId: string): void {
        for (let i = list.length - 1; i >= 0; i--) {
            if (list[i].id === deletedId) {
                list.splice(i, 1)
            }
        }
    }

    function onDeletedKyou(deletedKyou: Kyou): void {
        removeKyouFromListById(props.uploaded_kyous, deletedKyou.id)
        if (focused_kyou.value?.id === deletedKyou.id) {
            focused_kyou.value = null
        }
        emits('deleted_kyou', deletedKyou)
    }

    // ── Event relay objects ──
    const kyouListViewHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'focused_kyou': (kyou: Kyou) => focused_kyou.value = kyou,
        'clicked_kyou': (kyou: Kyou) => focused_kyou.value = kyou,
        'requested_reload_kyou': (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        'deleted_kyou': (kyou: Kyou) => onDeletedKyou(kyou),
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
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    const editIdfKyouViewHandlers = {
        'deleted_kyou': (kyou: Kyou) => onDeletedKyou(kyou),
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
        'requested_reload_kyou': (kyou: Kyou) => {
            reload_focused_kyou()
            emits('requested_reload_kyou', kyou)
        },
    }

    // ── Return ──
    return {
        // Template refs
        kyou_list_view,
        edit_idf_kyou_view,

        // State
        enable_context_menu,
        enable_dialog,
        focused_kyou,
        kyou_height,
        kyou_height_px,

        // Business logic
        reload_focused_kyou,
        onDeletedKyou,

        // Event relay objects
        kyouListViewHandlers,
        editIdfKyouViewHandlers,
    }
}
