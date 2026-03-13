import { computed, ref, type Ref } from 'vue'
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

export function useDnoteItemView(options: {
    props: DnoteItemProps,
    emits: DnoteItemViewEmits,
    model_value: Ref<DnoteItem | undefined>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const contextmenu = ref<any>(null)
    const confirm_delete_dnote_item_list_dialog = ref<any>(null)
    const edit_dnote_item_dialog = ref<any>(null)
    const kyou_list_view_dialog = ref<any>(null)

    // ── State refs ──
    const value = ref("")
    const related_kyous: Ref<Array<Kyou>> = ref([])
    const list_height = computed(() => (window.screen.height * 7) / 10)

    // ── Computed ──
    const aggregate_target_type = computed(() => model_value.value?.agregate_target?.to_json().type.toString() ?? "")
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
    ): Promise<any> {
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
    function onContextmenu(e: any): void {
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
        'deleted_kyou': (...kyou: any[]) => emits('deleted_kyou', kyou[0]),
        'deleted_tag': (...tag: any[]) => emits('deleted_tag', tag[0]),
        'deleted_text': (...text: any[]) => emits('deleted_text', text[0]),
        'deleted_notification': (...n: any[]) => emits('deleted_notification', n[0]),
        'registered_kyou': (...kyou: any[]) => emits('registered_kyou', kyou[0]),
        'registered_tag': (...tag: any[]) => emits('registered_tag', tag[0]),
        'registered_text': (...text: any[]) => emits('registered_text', text[0]),
        'registered_notification': (...n: any[]) => emits('registered_notification', n[0]),
        'updated_kyou': (...kyou: any[]) => emits('updated_kyou', kyou[0]),
        'updated_tag': (...tag: any[]) => emits('updated_tag', tag[0]),
        'updated_text': (...text: any[]) => emits('updated_text', text[0]),
        'updated_notification': (...n: any[]) => emits('updated_notification', n[0]),
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0]),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0]),
        'focused_kyou': (...kyou: any[]) => emits('focused_kyou', kyou[0] as Kyou),
        'clicked_kyou': (...kyou: any[]) => { emits('focused_kyou', kyou[0] as Kyou); emits('clicked_kyou', kyou[0] as Kyou) },
        'requested_open_rykv_dialog': (...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2]),
    }

    const contextMenuHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0]),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0]),
        'requested_delete_dnote_item_list': () => onRequestedDeleteDnoteItemList(),
        'requested_edit_dnote_item_list': () => onRequestedEditDnoteItemList(),
    }

    const confirmDeleteHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0]),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0]),
        'requested_delete_dnote_list_item': (...id: any[]) => emits('requested_delete_dnote_item', id[0] as string),
    }

    const editDnoteItemHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0]),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0]),
        'requested_update_dnote_item': (...d: any[]) => emits('requested_update_dnote_item', d[0]),
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
