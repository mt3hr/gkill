import { computed, ref, type Ref } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type DnoteItemProps from '@/pages/views/dnote-item-props'
import type DnoteItemViewEmits from '@/pages/views/dnote-item-view-emits'
import { DnoteAgregator } from '@/classes/dnote/dnote-aggregator'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type DnoteItem from '@/classes/dnote/dnote-item'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

export function useDnoteItemView(options: {
    props: DnoteItemProps,
    emits: DnoteItemViewEmits,
    model_value: Ref<DnoteItem | undefined>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const contextmenu = ref<ComponentRef | null>(null)
    const confirm_delete_dnote_item_list_dialog = ref<ComponentRef | null>(null)
    const edit_dnote_item_dialog = ref<ComponentRef | null>(null)
    const kyou_list_view_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const value = ref("")
    const related_kyous: Ref<Array<Kyou>> = ref([])
    const list_height = computed(() => (window.screen.height * 7) / 10)

    // ── Computed ──
    const aggregate_target_type = computed(() => (model_value.value?.agregate_target?.to_json().type as string)?.toString() ?? "")
    const is_lantana_type = computed(() => aggregate_target_type.value.includes("Lantana"))

    const is_plus_number_value = computed(() => {
        if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
            return !value.value.startsWith("-")
        }
        return false
    })
    const is_minus_number_value = computed(() => {
        if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
            return value.value.toString().startsWith("-")
        }
        return false
    })
    const value_class = computed(() => (is_plus_number_value.value ? "plus_value" : is_minus_number_value.value ? "minus_value" : ""))
    const mood_value = computed(() => Number(value.value).valueOf())

    // ── Business logic ──
    async function load_aggregated_value(
        abort_controller: AbortController,
        kyous: Array<Kyou>,
        query: FindKyouQuery,
        kyou_is_loaded: boolean
    ): Promise<void> {
        related_kyous.value.splice(0)
        const dnote_aggregator = new DnoteAgregator(model_value.value!.predicate, model_value.value!.agregate_target)
        const aggregate_result = await dnote_aggregator.agregate(abort_controller, kyous, query, kyou_is_loaded)
        value.value = aggregate_result.result_string.replace("<br>", "")
        related_kyous.value.splice(0, Infinity, ...aggregate_result.match_kyous)
        emits("finish_a_aggregate_task")
    }

    async function reset(): Promise<void> {
        value.value = ""
    }

    // ── DnD ──
    function drag_start(e: DragEvent): void {
        if (!props.editable) return
        const id = model_value.value?.id ?? ""
        if (!id) return

        if (e.dataTransfer) {
            e.dataTransfer.effectAllowed = "move"
            e.dataTransfer.setData("gkill_dnote_item_id", id)
            e.dataTransfer.setData("gkill_dnote_item_src_list_index", String(props.dnd_list_index))
        }
        e.stopPropagation()
    }

    function dragover(e: DragEvent): void {
        if (!props.editable) return
        if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
        e.preventDefault()
        e.stopPropagation()
    }

    function drop(e: DragEvent): void {
        if (!props.editable) return

        const srcId = e.dataTransfer?.getData("gkill_dnote_item_id")
        const srcListIndexStr = e.dataTransfer?.getData("gkill_dnote_item_src_list_index")
        const targetId = model_value.value?.id ?? ""
        if (!srcId || srcListIndexStr === undefined || srcListIndexStr === null || srcListIndexStr === "") return
        if (!targetId) return

        const srcListIndex = Number(srcListIndexStr)
        const targetListIndex = props.dnd_list_index
        if (srcId === targetId && srcListIndex === targetListIndex) return

        const el = e.currentTarget as HTMLElement | null
        if (!el) return
        const rect = el.getBoundingClientRect()
        const y = e.clientY - rect.top
        const dropType: "up" | "down" = y <= rect.height * 0.5 ? "up" : "down"

        emits("requested_move_dnote_item", srcId, srcListIndex, targetId, targetListIndex, dropType)
        e.preventDefault()
        e.stopPropagation()
    }

    // ── Template event handlers ──
    function onContextmenu(e: MouseEvent): void {
        if (props.editable) {
            contextmenu.value?.show(e, model_value.value!.id)
        }
    }

    function onDblclick(): void {
        kyou_list_view_dialog.value?.show()
    }

    function onRequestedDeleteDnoteItemList(): void {
        confirm_delete_dnote_item_list_dialog.value?.show(model_value.value!)
    }

    function onRequestedEditDnoteItemList(): void {
        edit_dnote_item_dialog.value?.show()
    }

    // ── CRUD relay handlers ──
    const kyouListViewDialogHandlers = {
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
        'focused_kyou': (kyou: Kyou) => emits('focused_kyou', kyou),
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    const contextMenuHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_delete_dnote_item_list': () => onRequestedDeleteDnoteItemList(),
        'requested_edit_dnote_item_list': () => onRequestedEditDnoteItemList(),
    }

    const confirmDeleteHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_delete_dnote_list_item': (value: string) => emits('requested_delete_dnote_item', value),
    }

    const editDnoteItemHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_update_dnote_item': (item: DnoteItem) => emits('requested_update_dnote_item', item),
    }

    return {
        // Template refs
        contextmenu,
        confirm_delete_dnote_item_list_dialog,
        edit_dnote_item_dialog,
        kyou_list_view_dialog,

        // State
        value,
        related_kyous,
        list_height,

        // Computed
        is_lantana_type,
        value_class,
        mood_value,

        // Business logic
        load_aggregated_value,
        reset,

        // DnD
        drag_start,
        dragover,
        drop,

        // Template event handlers
        onContextmenu,
        onDblclick,

        // Event relay objects
        kyouListViewDialogHandlers,
        contextMenuHandlers,
        confirmDeleteHandlers,
        editDnoteItemHandlers,
    }
}
