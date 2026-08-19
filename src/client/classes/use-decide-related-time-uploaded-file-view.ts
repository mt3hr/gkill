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
import { new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import { remove_kyou_from_list_by_id } from '@/classes/kyou-local-insert'

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
    // 引き直しは classes/kyou-reload.ts の手順を必ず通す。
    // 「SWキャッシュ削除 → reload(true) → is_typed_data_loaded=false → load_all(query, true)」の
    // 4手順のうち1つでも欠けると引き直しに失敗する。とくに load_all の force_attached を落とすと、
    // clone() が is_attached_tags_loaded を引き継ぐせいで load_attached_tags(false) が早期returnし、
    // 添付タグを一度も引き直さない（「タグを足しても表示が変わらない」の正体）。
    // requested_reload_kyou はタグ/テキスト/通知の変更の唯一の信号なので、ここが効かないと何も反映されない
    async function reload_kyou(kyou: Kyou): Promise<void> {
        // 1回の更新から派生する引き直しに同じ値を渡して1往復に畳む
        const requested_at = new_reload_batch()
        const refresh_focused = async (): Promise<void> => {
            if (!focused_kyou.value || focused_kyou.value.id !== kyou.id) {
                return
            }
            const refreshed = await refresh_kyou(focused_kyou.value, undefined, requested_at)
            if (refreshed) {
                focused_kyou.value = refreshed
            }
        }
        await Promise.all([
            refresh_kyou_in_list(props.uploaded_kyous, kyou, { requested_at: requested_at }),
            refresh_focused(),
        ])
    }

    function onDeletedKyou(deleted_kyou: Kyou): void {
        remove_kyou_from_list_by_id(props.uploaded_kyous, deleted_kyou.id)
        if (focused_kyou.value?.id === deleted_kyou.id) {
            focused_kyou.value = null
        }
        emits('deleted_kyou', deleted_kyou)
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
            reload_kyou(kyou)
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
        onDeletedKyou,

        // Event relay objects
        kyouListViewHandlers,
        editIdfKyouViewHandlers,
    }
}
