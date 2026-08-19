import { computed, nextTick, ref, type Ref } from "vue"
import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Kyou } from "@/classes/datas/kyou"
import type { ComponentRef } from "@/classes/component-ref"
import type { DnoteCorrelationCell, DnoteCorrelationGraphQuery, DnoteCorrelationPairPoint, DnoteCorrelationResult } from "@/classes/dnote/dnote-correlation"
import { DnoteCorrelationAggregator } from "@/classes/dnote/dnote-correlation-aggregator"
import { build_correlation_matrix_columns } from "@/classes/dnote-correlation-matrix-layout"
import type DnoteCorrelationGraphViewEmits from "@/pages/views/dnote-correlation-graph-view-emits"
import type DnoteCorrelationGraphViewProps from "@/pages/views/dnote-correlation-graph-view-props"
import { useDeviceKind } from '@/classes/use-device-kind'

export function useDnoteCorrelationGraphView(options: {
    props: DnoteCorrelationGraphViewProps,
    emits: DnoteCorrelationGraphViewEmits,
    model_value: Ref<DnoteCorrelationGraphQuery | undefined>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const contextmenu = ref<ComponentRef | null>(null)
    const confirm_delete_dnote_correlation_graph_dialog = ref<ComponentRef | null>(null)
    const edit_dnote_correlation_graph_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const result = ref<DnoteCorrelationResult | null>(null)
    const is_loading = ref(false)
    // 行列のどのセルを散布図に出すか。既定は「1つ目の指標 × 2つ目の指標」
    const selected_row_index = ref(0)
    const selected_column_index = ref(1)
    const selected_point = ref<DnoteCorrelationPairPoint | null>(null)

    // ── Computed ──
    const metrics = computed(() => model_value.value?.metrics ?? [])
    const selected_cell = computed<DnoteCorrelationCell | null>(() => result.value?.cells[selected_row_index.value]?.[selected_column_index.value] ?? null)
    const selected_row_metric = computed(() => metrics.value[selected_row_index.value] ?? null)
    const selected_column_metric = computed(() => metrics.value[selected_column_index.value] ?? null)
    // 1列目は指標名で幅を固定し、以降のセルは狭い列でも入る最小幅と、
    // 広い画面で間延びしない最大幅の間で伸縮させる。
    // 幅の根拠は classes/dnote-correlation-matrix-layout.ts のコメント参照
    const matrix_style = computed(() => ({ gridTemplateColumns: build_correlation_matrix_columns(metrics.value.length) }))

    // 散布図はviewBox 600x320の固定座標系。実データの範囲を軸の描画域へ線形写像する。
    // 全点が同値だと範囲0で除算が発散するので、±1に広げてから余白5%を足す
    const scatter_points = computed(() => {
        const points = selected_cell.value?.points ?? []
        if (points.length === 0) return []
        const xs = points.map(point => point.x)
        const ys = points.map(point => point.y)
        let x_min = Math.min(...xs)
        let x_max = Math.max(...xs)
        let y_min = Math.min(...ys)
        let y_max = Math.max(...ys)
        if (x_min === x_max) { x_min -= 1; x_max += 1 }
        if (y_min === y_max) { y_min -= 1; y_max += 1 }
        const x_padding = (x_max - x_min) * 0.05
        const y_padding = (y_max - y_min) * 0.05
        x_min -= x_padding
        x_max += x_padding
        y_min -= y_padding
        y_max += y_padding
        return points.map(point => ({
            point,
            cx: 52 + (point.x - x_min) / (x_max - x_min) * 524,
            cy: 272 - (point.y - y_min) / (y_max - y_min) * 244,
        }))
    })

    // ── Business logic ──
    async function load_correlation(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean): Promise<void> {
        if (!model_value.value) return
        is_loading.value = true
        try {
            result.value = await new DnoteCorrelationAggregator(model_value.value).aggregate(abort_controller, kyous, query, kyou_is_loaded)
            // 指標を減らして保存すると、選択中の列が範囲外に残る
            if (selected_column_index.value >= metrics.value.length) selected_column_index.value = Math.min(1, metrics.value.length - 1)
        } catch (_error: unknown) {
            // abort時は無視。Dnote全体の再検索や定義切替で中断された場合は、古い結果を表示しない
            result.value = null
        } finally {
            is_loading.value = false
            emits('finish_a_aggregate_task')
        }
    }

    async function reset(): Promise<void> {
        return nextTick(() => {
            result.value = null
            selected_point.value = null
        })
    }

    function select_cell(row_index: number, column_index: number): void {
        selected_row_index.value = row_index
        selected_column_index.value = column_index
        selected_point.value = null
    }

    // 相関の強さを不透明度に、符号を色に割り当てる。
    // 下限0.08は「相関ほぼ0のセル」と「算出不能(null)のセル」を見分けるための最低限の着色
    function heatmap_color(cell: DnoteCorrelationCell): string {
        if (cell.coefficient === null) return "rgb(var(--v-theme-surface))"
        if (cell.coefficient === 0) return "rgb(var(--v-theme-surface))"
        const opacity = 0.08 + Math.abs(cell.coefficient) * 0.82
        const color = cell.coefficient < 0 ? "--v-theme-error" : "--v-theme-primary"
        return `rgba(var(${color}), ${opacity})`
    }

    function heatmap_text_color(cell: DnoteCorrelationCell): string {
        return cell.coefficient !== null && Math.abs(cell.coefficient) >= 0.55
            ? "rgb(var(--v-theme-on-primary))"
            : "rgb(var(--v-theme-on-surface))"
    }

    // ── DnD ──
    // ドラッグ&ドロップでの並べ替えはPCでのみ有効にする。
    // タッチ端末では draggable を立てても掴めないうえ、
    // 立てたままだとスクロールやロングプレスと競合する。
    // 判定は useDeviceKind の is_pc（"タッチできるか"ではない）。
    // 同じ形が use-foldable-struct.ts / use-mi-kyou-view.ts / use-mi-re-kyou-view.ts にある
    const { is_pc } = useDeviceKind()
    const effective_draggable = computed(() => is_pc.value && props.editable)

    function drag_start(event: DragEvent): void {
        if (!effective_draggable.value || !model_value.value?.id) return
        event.dataTransfer?.setData("gkill_dnote_correlation_graph_id", model_value.value.id)
        if (event.dataTransfer) event.dataTransfer.effectAllowed = "move"
        event.stopPropagation()
    }

    function dragover(event: DragEvent): void {
        if (!effective_draggable.value) return
        event.preventDefault()
        if (event.dataTransfer) event.dataTransfer.dropEffect = "move"
    }

    function drop(event: DragEvent): void {
        if (!effective_draggable.value || !model_value.value) return
        const source_id = event.dataTransfer?.getData("gkill_dnote_correlation_graph_id")
        if (!source_id || source_id === model_value.value.id) return
        const element = event.currentTarget as HTMLElement | null
        if (!element) return
        // 縦積みのため上下で前後を判定する（'left'=前、'right'=後）
        const rect = element.getBoundingClientRect()
        const direction = event.clientY - rect.top <= rect.height / 2 ? "left" : "right"
        emits('requested_move_dnote_correlation_graph', source_id, model_value.value.id, direction)
        event.preventDefault()
        event.stopPropagation()
    }

    // ── Template event handlers ──
    function onContextmenu(e: MouseEvent): void {
        if (props.editable) {
            contextmenu.value?.show(e, model_value.value!.id)
        }
    }

    function onRequestedDeleteDnoteCorrelationGraph(): void {
        confirm_delete_dnote_correlation_graph_dialog.value?.show(model_value.value!)
    }

    function onRequestedEditDnoteCorrelationGraph(): void {
        if (!props.editable) return
        edit_dnote_correlation_graph_dialog.value?.show()
    }

    // ── Event relay objects ──
    const contextMenuHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_delete_dnote_correlation_graph': () => onRequestedDeleteDnoteCorrelationGraph(),
        'requested_edit_dnote_correlation_graph': () => onRequestedEditDnoteCorrelationGraph(),
    }

    const confirmDeleteHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_delete_dnote_correlation_graph': (value: string) => emits('requested_delete_dnote_correlation_graph', value),
    }

    const editDnoteCorrelationGraphHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_update_dnote_correlation_graph': (query: DnoteCorrelationGraphQuery) => emits('requested_update_dnote_correlation_graph', query),
    }

    // ── Return ──
    return {
        // DnD
        effective_draggable,
        // Template refs
        contextmenu,
        confirm_delete_dnote_correlation_graph_dialog,
        edit_dnote_correlation_graph_dialog,

        // State
        result,
        is_loading,
        selected_row_index,
        selected_column_index,
        selected_point,

        // Computed
        metrics,
        selected_cell,
        selected_row_metric,
        selected_column_metric,
        matrix_style,
        scatter_points,

        // Business logic
        load_correlation,
        reset,
        select_cell,
        heatmap_color,
        heatmap_text_color,

        // DnD
        drag_start,
        dragover,
        drop,

        // Template event handlers
        onContextmenu,
        onRequestedDeleteDnoteCorrelationGraph,
        onRequestedEditDnoteCorrelationGraph,

        // Event relay objects
        contextMenuHandlers,
        confirmDeleteHandlers,
        editDnoteCorrelationGraphHandlers,
    }
}
