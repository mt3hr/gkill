import { i18n } from '@/i18n'
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

export function useDnoteListView(options: {
    props: DnoteListViewProps,
    emits: DnoteListViewEmits,
    model_value: Ref<DnoteListQuery | undefined>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const list_view = ref<any>(null)
    const contextmenu = ref<any>(null)
    const confirm_delete_dnote_list_query_dialog = ref<any>(null)
    const edit_dnote_list_query = ref<any>(null)

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
    function onContextmenu(e: any): void {
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
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
    }

    const aggregatedListItemHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
        'focused_kyou': (...kyou: any[]) => emits('focused_kyou', kyou[0] as Kyou),
        'clicked_kyou': (...kyou: any[]) => { emits('focused_kyou', kyou[0] as Kyou); emits('clicked_kyou', kyou[0] as Kyou) },
        'requested_delete_dnote_list_query': (...id: any[]) => emits('requested_delete_dnote_list_query', id[0] as string),
        'requested_update_dnote_list_query': (...dnote_list_query: any[]) => emits('requested_update_dnote_list_query', dnote_list_query[0] as DnoteListQuery),
        'deleted_kyou': (...kyou: any[]) => emits('deleted_kyou', kyou[0] as Kyou),
        'deleted_tag': (...tag: any[]) => emits('deleted_tag', tag[0] as Tag),
        'deleted_text': (...text: any[]) => emits('deleted_text', text[0] as Text),
        'deleted_notification': (...notification: any[]) => emits('deleted_notification', notification[0] as Notification),
        'registered_kyou': (...kyou: any[]) => emits('registered_kyou', kyou[0] as Kyou),
        'registered_tag': (...tag: any[]) => emits('registered_tag', tag[0] as Tag),
        'registered_text': (...text: any[]) => emits('registered_text', text[0] as Text),
        'registered_notification': (...notification: any[]) => emits('registered_notification', notification[0] as Notification),
        'updated_kyou': (...kyou: any[]) => emits('updated_kyou', kyou[0] as Kyou),
        'updated_tag': (...tag: any[]) => emits('updated_tag', tag[0] as Tag),
        'updated_text': (...text: any[]) => emits('updated_text', text[0] as Text),
        'updated_notification': (...notification: any[]) => emits('updated_notification', notification[0] as Notification),
        'requested_open_rykv_dialog': (...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2]),
    }

    const contextMenuHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
        'requested_delete_dnote_list_query': () => onRequestedDeleteDnoteListQuery(),
        'requested_edit_dnote_list_query': () => onRequestedEditDnoteListQuery(),
    }

    const confirmDeleteHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
        'requested_delete_dnote_list_query': (...id: any[]) => emits('requested_delete_dnote_list_query', id[0] as string),
    }

    const editDnoteListHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
        'requested_update_dnote_list_query': (...dnote_list_query: any[]) => emits('requested_update_dnote_list_query', dnote_list_query[0] as DnoteListQuery),
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
