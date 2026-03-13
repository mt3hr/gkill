import { computed, type Ref, ref } from 'vue'
import type { DecideRelatedTimeUploadedFileViewProps } from '@/pages/views/decide-related-time-uploaded-file-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'

export function useDecideRelatedTimeUploadedFileView(options: {
    props: DecideRelatedTimeUploadedFileViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_list_view = ref<any>(null)
    const edit_idf_kyou_view = ref<any>(null)

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
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
        'focused_kyou': (...kyou: any[]) => focused_kyou.value = kyou[0] as Kyou,
        'clicked_kyou': (...kyou: any[]) => focused_kyou.value = kyou[0] as Kyou,
        'requested_reload_kyou': (...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou),
        'deleted_kyou': (...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou),
        'deleted_tag': (...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag),
        'deleted_text': (...deleted_text: any[]) => emits('deleted_text', deleted_text[0]),
        'deleted_notification': (...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification),
        'registered_kyou': (...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou),
        'registered_tag': (...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag),
        'registered_text': (...registered_text: any[]) => emits('registered_text', registered_text[0] as Text),
        'registered_notification': (...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification),
        'updated_kyou': (...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou),
        'updated_tag': (...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag),
        'updated_text': (...updated_text: any[]) => emits('updated_text', updated_text[0] as Text),
        'updated_notification': (...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification),
        'requested_open_rykv_dialog': (...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2]),
    }

    const editIdfKyouViewHandlers = {
        'deleted_kyou': (...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou),
        'deleted_tag': (...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag),
        'deleted_text': (...deleted_text: any[]) => emits('deleted_text', deleted_text[0]),
        'deleted_notification': (...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification),
        'registered_kyou': (...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou),
        'registered_tag': (...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag),
        'registered_text': (...registered_text: any[]) => emits('registered_text', registered_text[0] as Text),
        'registered_notification': (...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification),
        'updated_kyou': (...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou),
        'updated_tag': (...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag),
        'updated_text': (...updated_text: any[]) => emits('updated_text', updated_text[0] as Text),
        'updated_notification': (...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification),
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'requested_reload_kyou': (...kyou: any[]) => {
            reload_focused_kyou()
            emits('requested_reload_kyou', kyou[0] as Kyou)
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
