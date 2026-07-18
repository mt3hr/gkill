import { computed, nextTick, type Ref, ref } from 'vue'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import { DnoteTrendAggregator } from '@/classes/dnote/dnote-trend-aggregator'
import type DnoteTrendPoint from '@/classes/dnote/dnote-trend/dnote-trend-point'
import type DnoteTrendGraphViewProps from '@/pages/views/dnote-trend-graph-view-props'
import type DnoteTrendGraphQuery from '@/pages/views/dnote-trend-graph-query'
import type DnoteTrendGraphViewEmits from '@/pages/views/dnote-trend-graph-view-emits'
import type { ComponentRef } from '@/classes/component-ref'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

export function useDnoteTrendGraphView(options: {
    props: DnoteTrendGraphViewProps,
    emits: DnoteTrendGraphViewEmits,
    model_value: Ref<DnoteTrendGraphQuery | undefined>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const contextmenu = ref<ComponentRef | null>(null)
    const confirm_delete_dnote_trend_graph_dialog = ref<ComponentRef | null>(null)
    const edit_dnote_trend_graph_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const trend_points: Ref<Array<DnoteTrendPoint>> = ref(new Array<DnoteTrendPoint>())
    const is_loading = ref(false)

    // ── Computed ──
    const sparkline_type = computed(() => model_value.value?.chart_type === 'bar' ? 'bar' : 'trend')

    const sparkline_values = computed(() => trend_points.value.map((point) => point.value))

    // ラベルは間引き表示（先頭・末尾は必ず表示。末尾直前は重なるため省略）
    // VSparklineはfalsyなラベルを値にフォールバックするため、非表示位置は空白文字にする
    const sparkline_labels = computed(() => {
        const points = trend_points.value
        // 1点のみの場合、VSparklineは点を複製して2点にするため2つ目のラベルを埋める
        if (points.length === 1) {
            return [points[0].label, " "]
        }
        const stride = Math.ceil(points.length / 6)
        return points.map((point, i) => {
            if (i === points.length - 1) return point.label
            if (i % stride === 0 && points.length - 1 - i >= stride) return point.label
            return " "
        })
    })

    // 件数・合計系は0起点、平均系はオートスケール
    const sparkline_min = computed(() => {
        const type = model_value.value?.aggregate_target.to_json().type
        return (typeof type === 'string' && (type.startsWith('AgregateCount') || type.startsWith('AgregateSum'))) ? 0 : undefined
    })

    const sparkline_tooltip = computed(() => ({
        titleFormat: (item: { index: number, value: number }) => {
            const point = trend_points.value[item.index]
            if (!point) return ""
            const value_string = point.value_string !== "" ? point.value_string : point.value.toString()
            return `${point.label}: ${value_string}`
        },
    }))

    const is_all_empty = computed(() => trend_points.value.length === 0 || trend_points.value.every((point) => point.match_kyous.length === 0))

    // ── Business logic ──
    async function load_trend_graph(
        abort_controller: AbortController,
        kyous: Array<Kyou>,
        find_kyou_query: FindKyouQuery,
        kyou_is_loaded: boolean
    ): Promise<void> {
        if (!model_value.value) return
        is_loading.value = true
        try {
            const aggregator = new DnoteTrendAggregator(
                model_value.value.predicate,
                model_value.value.aggregate_target,
                model_value.value.granularity
            )
            const points = await aggregator.aggregate_trend(abort_controller, kyous, find_kyou_query, kyou_is_loaded)

            trend_points.value.splice(0)
            for (let i = 0; i < points.length; i++) {
                trend_points.value.push(points[i])
            }
        } catch (_e: unknown) {
            // abort時は無視
        } finally {
            is_loading.value = false
            emits('finish_a_aggregate_task')
        }
    }

    async function reset(): Promise<void> {
        return nextTick(async () => {
            trend_points.value.splice(0)
        })
    }

    // ── DnD ──
    type DropType = 'left' | 'right'

    function drag_start(e: DragEvent): void {
        if (!props.editable) return
        const id = model_value.value?.id ?? ''
        if (!id) return

        e.dataTransfer?.setData('gkill_dnote_trend_graph_id', id)
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

        const srcId = e.dataTransfer?.getData('gkill_dnote_trend_graph_id')
        const targetId = model_value.value?.id ?? ''
        if (!srcId || !targetId) return
        if (srcId === targetId) return

        const el = e.currentTarget as HTMLElement | null
        if (!el) return

        // 縦積みのため上下で前後を判定する（'left'=前、'right'=後）
        const rect = el.getBoundingClientRect()
        const y = e.clientY - rect.top
        const dropType: DropType = (y <= rect.height * 0.5) ? 'left' : 'right'

        emits('requested_move_dnote_trend_graph', srcId, targetId, dropType)

        e.preventDefault()
        e.stopPropagation()
    }

    // ── Template event handlers ──
    function onContextmenu(e: MouseEvent): void {
        if (props.editable) {
            contextmenu.value?.show(e, model_value.value!.id)
        }
    }

    function onRequestedDeleteDnoteTrendGraph(): void {
        confirm_delete_dnote_trend_graph_dialog.value?.show(model_value.value!)
    }

    function onRequestedEditDnoteTrendGraph(): void {
        edit_dnote_trend_graph_dialog.value?.show()
    }

    // ── Event relay objects ──
    const contextMenuHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_delete_dnote_trend_graph': () => onRequestedDeleteDnoteTrendGraph(),
        'requested_edit_dnote_trend_graph': () => onRequestedEditDnoteTrendGraph(),
    }

    const confirmDeleteHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_delete_dnote_trend_graph': (value: string) => emits('requested_delete_dnote_trend_graph', value),
    }

    const editDnoteTrendGraphHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_update_dnote_trend_graph': (query: DnoteTrendGraphQuery) => emits('requested_update_dnote_trend_graph', query),
    }

    return {
        // Template refs
        contextmenu,
        confirm_delete_dnote_trend_graph_dialog,
        edit_dnote_trend_graph_dialog,

        // State
        trend_points,
        is_loading,

        // Computed
        sparkline_type,
        sparkline_values,
        sparkline_labels,
        sparkline_min,
        sparkline_tooltip,
        is_all_empty,

        // Business logic
        load_trend_graph,
        reset,

        // DnD
        drag_start,
        dragover,
        drop,

        // Template event handlers
        onContextmenu,
        onRequestedDeleteDnoteTrendGraph,
        onRequestedEditDnoteTrendGraph,

        // Event relay objects
        contextMenuHandlers,
        confirmDeleteHandlers,
        editDnoteTrendGraphHandlers,
    }
}
