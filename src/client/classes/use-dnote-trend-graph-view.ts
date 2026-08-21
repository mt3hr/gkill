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
import { format_day_of_week, to_single_line } from '@/classes/format-date-time'
import { useDeviceKind } from '@/classes/use-device-kind'

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
    const is_zero_based = computed(() => {
        const type = model_value.value?.aggregate_target.to_json().type
        return typeof type === 'string' && (type.startsWith('AggregateCount') || type.startsWith('AggregateSum'))
    })

    // 0起点でも、出費のように値が負になりうる系列では0を下限にすると
    // VSparkline内部のスケールが反転・発散して描画座標が領域外に飛ぶ。
    // 0は「必ず含める」だけにとどめ、実データの範囲は常に包含させる
    const sparkline_min = computed(() => {
        if (!is_zero_based.value) return undefined
        const values = sparkline_values.value
        if (values.length === 0) return 0
        return Math.min(0, ...values)
    })

    const sparkline_max = computed(() => {
        if (!is_zero_based.value) return undefined
        const values = sparkline_values.value
        if (values.length === 0) return 0
        return Math.max(0, ...values)
    })

    const sparkline_tooltip = computed(() => ({
        titleFormat: (item: { index: number, value: number }) => {
            // VSparklineは1点のとき点を複製するため、indexが範囲外になり得る。クランプして必ず表示する
            const point = trend_points.value[Math.min(item.index, trend_points.value.length - 1)]
            if (!point) return ""
            const value_string = to_single_line(point.value_string !== "" ? point.value_string : point.value.toString())
            let label = point.label
            // 日単位のときは曜日も表示する
            if (model_value.value?.granularity === 'day') {
                const [year, month, date] = point.bucket_key.split('-').map(Number)
                label = `${point.label}(${format_day_of_week(new Date(year, month - 1, date))})`
            }
            return `${label}: ${value_string}`
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
    // ドラッグ&ドロップでの並べ替えはPCでのみ有効にする。
    // タッチ端末では draggable を立てても掴めないうえ、
    // 立てたままだとスクロールやロングプレスと競合する。
    // 判定は useDeviceKind の is_pc（"タッチできるか"ではない）。
    // 同じ形が use-foldable-struct.ts / use-mi-kyou-view.ts / use-mi-re-kyou-view.ts にある
    const { is_pc } = useDeviceKind()
    const effective_draggable = computed(() => is_pc.value && props.editable)

    type DropType = 'left' | 'right'

    function drag_start(e: DragEvent): void {
        if (!effective_draggable.value) return
        const id = model_value.value?.id ?? ''
        if (!id) return

        e.dataTransfer?.setData('gkill_dnote_trend_graph_id', id)
        if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
        e.stopPropagation()
    }

    function dragover(e: DragEvent): void {
        if (!effective_draggable.value) return
        if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
        e.preventDefault()
    }

    function drop(e: DragEvent): void {
        if (!effective_draggable.value) return

        const src_id = e.dataTransfer?.getData('gkill_dnote_trend_graph_id')
        const target_id = model_value.value?.id ?? ''
        if (!src_id || !target_id) return
        if (src_id === target_id) return

        const el = e.currentTarget as HTMLElement | null
        if (!el) return

        // 縦積みのため上下で前後を判定する（'left'=前、'right'=後）
        const rect = el.getBoundingClientRect()
        const y = e.clientY - rect.top
        const drop_type: DropType = (y <= rect.height * 0.5) ? 'left' : 'right'

        emits('requested_move_dnote_trend_graph', src_id, target_id, drop_type)

        e.preventDefault()
        e.stopPropagation()
    }

    // ── Template event handlers ──
    // VSparklineはマウスイベントのみ対応で、タッチ端末では合成mousemoveが抑制されることがある。
    // さらに表示中のVTooltipはoutside-click(2回目以降のタップ)で内部状態だけが閉じ、
    // propのtooltipVisibleがtrueのまま食い違うと以降再表示されない。
    // そのためmouseleave→mousemoveの順で疑似イベントを送り、閉→開のサイクルを強制して毎回確実に表示する
    function dispatch_tooltip_cycle(root: HTMLElement | null, client_x: number, client_y: number): void {
        const svg = root?.querySelector('svg')
        if (!svg) return
        svg.dispatchEvent(new MouseEvent('mouseleave'))
        svg.dispatchEvent(new MouseEvent('mousemove', { clientX: client_x, clientY: client_y }))
    }

    const touch_start_point = { x: 0, y: 0, valid: false }

    function onGraphTouchstart(e: TouchEvent): void {
        if (e.touches.length !== 1) {
            touch_start_point.valid = false
            return
        }
        touch_start_point.x = e.touches[0].clientX
        touch_start_point.y = e.touches[0].clientY
        touch_start_point.valid = true
    }

    function onGraphTouchend(e: TouchEvent): void {
        if (!touch_start_point.valid || e.changedTouches.length !== 1) return
        touch_start_point.valid = false
        const touch = e.changedTouches[0]
        // 指が動いていたらスクロール等の操作なので何もしない
        if (Math.hypot(touch.clientX - touch_start_point.x, touch.clientY - touch_start_point.y) > 10) return
        // 合成マウスイベント(click等)による二重処理を防ぐ
        e.preventDefault()
        dispatch_tooltip_cycle(e.currentTarget as HTMLElement | null, touch.clientX, touch.clientY)
    }

    function onGraphClick(e: MouseEvent): void {
        dispatch_tooltip_cycle(e.currentTarget as HTMLElement | null, e.clientX, e.clientY)
    }

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
        // DnD
        effective_draggable,
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
        sparkline_max,
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
        onGraphTouchstart,
        onGraphTouchend,
        onGraphClick,
        onContextmenu,
        onRequestedDeleteDnoteTrendGraph,
        onRequestedEditDnoteTrendGraph,

        // Event relay objects
        contextMenuHandlers,
        confirmDeleteHandlers,
        editDnoteTrendGraphHandlers,
    }
}
