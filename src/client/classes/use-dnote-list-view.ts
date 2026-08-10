import { nextTick, type Ref, ref } from 'vue'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type AggregatedItem from '@/classes/dnote/aggregate-grouping-list-result-record'
import { DnoteListAggregator } from '@/classes/dnote/dnote-list-aggregator'
import type DnoteListViewProps from '@/pages/views/dnote-list-view-props'
import type DnoteListQuery from '@/pages/views/dnote-list-query'
import type DnoteListViewEmits from '@/pages/views/dnote-list-view-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'

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
    const aggregated_items: Ref<Array<AggregatedItem>> = ref(new Array<AggregatedItem>())

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

        const src_id = e.dataTransfer?.getData('gkill_dnote_list_id')
        const target_id = model_value.value?.id ?? ''
        if (!src_id || !target_id) return
        if (src_id === target_id) return

        const el = e.currentTarget as HTMLElement | null
        if (!el) return

        const rect = el.getBoundingClientRect()
        const x = e.clientX - rect.left
        const drop_type: DropType = (x <= rect.width * 0.5) ? 'left' : 'right'

        emits('requested_move_dnote_list_query', src_id, target_id, drop_type)

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

    // 手書きで並べていた頃は requested_reload_kyou / requested_reload_list /
    // requested_update_check_kyous の3つを落としていた。
    // AggregatedListItem 側は全20件を出しているのに、ここで捨てられていた
    const aggregatedListItemHandlers = {
        ...build_kyou_dialog_relay(emits, {
            // クリックはフォーカス移動も伴う
            'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
        }),
        // Dnote固有のイベントは共通束に含まれない
        'requested_delete_dnote_list_query': (value: string) => emits('requested_delete_dnote_list_query', value),
        'requested_update_dnote_list_query': (query: DnoteListQuery) => emits('requested_update_dnote_list_query', query),
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
