import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { nextTick, type Ref, ref } from 'vue'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type AgregatedItem from '@/classes/dnote/aggregate-grouping-list-result-record'
import { DnoteListAggregator } from '@/classes/dnote/dnote-list-aggregator'
import type DnoteListViewProps from '@/pages/views/dnote-list-view-props'
import type DnoteListQuery from '@/pages/views/dnote-list-query'
import type DnoteListViewEmits from '@/pages/views/dnote-list-view-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

export function useDnoteListView(options: {
    props: DnoteListViewProps,
    emits: DnoteListViewEmits,
    model_value: Ref<DnoteListQuery | undefined>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const list_view = ref<ComponentRef | null>(null)
    const contextmenu = ref<ComponentRef | null>(null)
    const confirm_delete_dnote_list_query_dialog = ref<ComponentRef | null>(null)
    const edit_dnote_list_query = ref<ComponentRef | null>(null)

    // ── State refs ──
    const aggregated_items: Ref<Array<AgregatedItem>> = ref(new Array<AgregatedItem>())

    // ── Business logic ──
    async function load_aggregate_grouping_list(
        abort_controller: AbortController,
        kyous: Array<Kyou>,
        find_kyou_query: FindKyouQuery,
        kyou_is_loaded: boolean
    ): Promise<void> {
        if (!model_value.value) return

        const list_aggregator = new DnoteListAggregator(
            model_value.value.predicate,
            model_value.value.key_getter,
            model_value.value.aggregate_target
        )
        const aggregated_result = await list_aggregator.aggregate_grouping_list(
            abort_controller,
            kyous,
            find_kyou_query,
            kyou_is_loaded
        )

        aggregated_items.value.splice(0)
        for (let i = 0; i < aggregated_result.length; i++) {
            aggregated_items.value.push(aggregated_result[i])
        }
        emits('finish_a_aggregate_task')
    }

    async function reset(): Promise<void> {
        return nextTick(async () => {
            aggregated_items.value.splice(0)
        })
    }

    // ── DnD ──
    type DropType = 'left' | 'right'

    function drag_start(e: DragEvent): void {
        if (!props.editable) return
        const id = model_value.value?.id ?? ''
        if (!id) return

        e.dataTransfer?.setData('gkill_dnote_list_id', id)
        if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
        e.stopPropagation()
    }

    function dragover(e: DragEvent): void {
        if (!props.editable) return
        if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
        e.preventDefault()
    }

    function drop(e: DragEvent): void {
        if (!props.editable) return

        const srcId = e.dataTransfer?.getData('gkill_dnote_list_id')
        const targetId = model_value.value?.id ?? ''
        if (!srcId || !targetId) return
        if (srcId === targetId) return

        const el = e.currentTarget as HTMLElement | null
        if (!el) return

        const rect = el.getBoundingClientRect()
        const x = e.clientX - rect.left
        const dropType: DropType = (x <= rect.width * 0.5) ? 'left' : 'right'

        emits('requested_move_dnote_list_query', srcId, targetId, dropType)

        e.preventDefault()
        e.stopPropagation()
    }

    // ── Template event handlers ──
    function onContextmenu(e: MouseEvent): void {
        if (props.editable) {
            contextmenu.value?.show(e, model_value.value!.id)
        }
    }

    function onRequestedDeleteDnoteListQuery(): void {
        confirm_delete_dnote_list_query_dialog.value?.show(model_value.value!)
    }

    function onRequestedEditDnoteListQuery(): void {
        edit_dnote_list_query.value?.show()
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    const aggregatedListItemHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'focused_kyou': (kyou: Kyou) => emits('focused_kyou', kyou),
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
        'requested_delete_dnote_list_query': (value: string) => emits('requested_delete_dnote_list_query', value),
        'requested_update_dnote_list_query': (query: DnoteListQuery) => emits('requested_update_dnote_list_query', query),
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
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    const contextMenuHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_delete_dnote_list_query': () => onRequestedDeleteDnoteListQuery(),
        'requested_edit_dnote_list_query': () => onRequestedEditDnoteListQuery(),
    }

    const confirmDeleteHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_delete_dnote_list_query': (value: string) => emits('requested_delete_dnote_list_query', value),
    }

    const editDnoteListHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_update_dnote_list_query': (query: DnoteListQuery) => emits('requested_update_dnote_list_query', query),
    }

    return {
        // Template refs
        list_view,
        contextmenu,
        confirm_delete_dnote_list_query_dialog,
        edit_dnote_list_query,

        // State
        aggregated_items,

        // Business logic
        load_aggregate_grouping_list,
        reset,

        // DnD
        drag_start,
        dragover,
        drop,

        // Template event handlers
        onContextmenu,
        onRequestedDeleteDnoteListQuery,
        onRequestedEditDnoteListQuery,

        // Event relay objects
        crudRelayHandlers,
        aggregatedListItemHandlers,
        contextMenuHandlers,
        confirmDeleteHandlers,
        editDnoteListHandlers,
    }
}
