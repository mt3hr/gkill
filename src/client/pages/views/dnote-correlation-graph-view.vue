<template>
  <section class="dnote_correlation_graph" @dragover="dragover" @drop="drop" @contextmenu.prevent.stop="onContextmenu">
    <!-- ドラッグはタイトルをハンドルにする（スクロール操作と干渉しにくい） -->
    <h2 :draggable="editable" :class="{ draggable: editable }" @dragstart="drag_start"
      @dblclick="onRequestedEditDnoteCorrelationGraph">{{ model_value?.title }}</h2>
    <v-progress-linear v-if="is_loading" indeterminate color="primary" />
    <template v-else-if="result">
      <!-- 指標を行と列に並べた総当たりのヒートマップ。セルを押すと下の散布図が切り替わる -->
      <div class="correlation_matrix_scroll">
        <div class="correlation_matrix" :style="matrix_style" role="grid">
          <div class="matrix_corner" />
          <div v-for="metric in metrics" :key="`column-${metric.id}`" class="matrix_header column_header">{{ metric.title }}</div>
          <template v-for="(row_metric, row_index) in metrics" :key="`row-${row_metric.id}`">
            <div class="matrix_header row_header">{{ row_metric.title }}</div>
            <button v-for="(cell, column_index) in result.cells[row_index]" :key="cell.column_metric_id"
              type="button" class="matrix_cell" role="gridcell"
              :class="{ selected: row_index === selected_row_index && column_index === selected_column_index }"
              :style="{ backgroundColor: heatmap_color(cell), color: heatmap_text_color(cell) }"
              :aria-label="`${row_metric.title} / ${metrics[column_index].title}: ${cell.coefficient === null ? '—' : cell.coefficient.toFixed(3)}, n=${cell.sample_size}`"
              @click="select_cell(row_index, column_index)">
              <span>{{ cell.coefficient === null ? '—' : cell.coefficient.toFixed(2) }}</span>
              <small>n={{ cell.sample_size }}</small>
            </button>
          </template>
        </div>
      </div>

      <v-card v-if="selected_cell && selected_row_metric && selected_column_metric" variant="outlined" class="pa-2 mt-2">
        <h3>{{ selected_row_metric.title }} → {{ selected_column_metric.title }}</h3>
        <p>
          {{ i18n.global.t('DNOTE_CORRELATION_COEFFICIENT_TITLE') }}:
          {{ selected_cell.coefficient === null ? '—' : selected_cell.coefficient.toFixed(4) }} / n={{ selected_cell.sample_size }}
        </p>
        <p>
          p={{ format_stat(selected_cell.p_value) }} /
          {{ i18n.global.t('DNOTE_CORRELATION_CONFIDENCE_INTERVAL_TITLE') }}:
          {{ format_interval(selected_cell.confidence_low, selected_cell.confidence_high) }}
        </p>
        <p v-if="model_value?.method === 'spearman'" class="text-medium-emphasis">
          {{ i18n.global.t('DNOTE_CORRELATION_SPEARMAN_APPROXIMATION_NOTE') }}
        </p>
        <p class="text-medium-emphasis">{{ i18n.global.t('DNOTE_CORRELATION_CAUSATION_NOTE') }}</p>

        <!-- viewBoxは固定座標系。実データはuseDnoteCorrelationGraphView側でこの座標へ写像済み -->
        <svg v-if="scatter_points.length > 0" class="correlation_scatter" viewBox="0 0 600 320"
          role="img" :aria-label="i18n.global.t('DNOTE_CORRELATION_SCATTER_TITLE')">
          <line x1="52" y1="272" x2="576" y2="272" class="scatter_axis" />
          <line x1="52" y1="28" x2="52" y2="272" class="scatter_axis" />
          <text x="314" y="312" text-anchor="middle" class="scatter_label">{{ selected_row_metric.title }}</text>
          <text x="14" y="150" text-anchor="middle" class="scatter_label" transform="rotate(-90 14 150)">{{ selected_column_metric.title }}</text>
          <circle v-for="scatter in scatter_points" :key="`${scatter.point.row_bucket_key}-${scatter.point.column_bucket_key}`"
            :cx="scatter.cx" :cy="scatter.cy" r="5" class="scatter_point" tabindex="0"
            @mouseenter="selected_point = scatter.point" @focus="selected_point = scatter.point"
            @click.stop="selected_point = scatter.point">
            <title>{{ point_description(scatter.point) }}</title>
          </circle>
        </svg>
        <div v-else class="text-center text-grey">{{ i18n.global.t('NO_RESULTS_MESSAGE') }}</div>
        <p v-if="selected_point" class="scatter_point_detail">{{ point_description(selected_point) }}</p>
      </v-card>
    </template>

    <DnoteCorrelationGraphContextMenu :application_config="application_config" :gkill_api="gkill_api"
      v-on="contextMenuHandlers" ref="contextmenu" />

    <ConfirmDeleteDnoteCorrelationGraphDialog :application_config="application_config" :gkill_api="gkill_api"
      v-on="confirmDeleteHandlers" ref="confirm_delete_dnote_correlation_graph_dialog" />

    <EditDnoteCorrelationGraphDialog :application_config="application_config" :gkill_api="gkill_api"
      :dnote_correlation_graph_query="model_value!" v-on="editDnoteCorrelationGraphHandlers"
      ref="edit_dnote_correlation_graph_dialog" />
  </section>
</template>

<script setup lang="ts">
import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { Kyou } from "@/classes/datas/kyou"
import type { DnoteCorrelationGraphQuery, DnoteCorrelationPairPoint } from "@/classes/dnote/dnote-correlation"
import type DnoteCorrelationGraphViewEmits from "./dnote-correlation-graph-view-emits"
import type DnoteCorrelationGraphViewProps from "./dnote-correlation-graph-view-props"
import DnoteCorrelationGraphContextMenu from "./dnote-correlation-graph-context-menu.vue"
import EditDnoteCorrelationGraphDialog from "../dialogs/edit-dnote-correlation-graph-dialog.vue"
import ConfirmDeleteDnoteCorrelationGraphDialog from "../dialogs/confirm-delete-dnote-correlation-graph-dialog.vue"
import { i18n } from "@/i18n"
import { useDnoteCorrelationGraphView } from "@/classes/use-dnote-correlation-graph-view"

const props = defineProps<DnoteCorrelationGraphViewProps>()
const emits = defineEmits<DnoteCorrelationGraphViewEmits>()
const model_value = defineModel<DnoteCorrelationGraphQuery>()

const {
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
  onRequestedEditDnoteCorrelationGraph,

  // Event relay objects
  contextMenuHandlers,
  confirmDeleteHandlers,
  editDnoteCorrelationGraphHandlers,
} = useDnoteCorrelationGraphView({ props, emits, model_value })

// p値は0付近で有効数字が飛ぶので、極小のときだけ指数表記に切り替える
function format_stat(value: number | null): string {
  if (value === null) return "—"
  if (value === 0) return "<0.0001"
  return value < 0.0001 ? value.toExponential(2) : value.toFixed(4)
}

function format_interval(low: number | null, high: number | null): string {
  return low === null || high === null ? "—" : `[${low.toFixed(4)}, ${high.toFixed(4)}]`
}

function point_description(point: DnoteCorrelationPairPoint): string {
  const x = point.x_value_string || point.x.toString()
  const y = point.y_value_string || point.y.toString()
  return `${point.row_label}: ${x} / ${point.column_label}: ${y}`
}

defineExpose({
  load_correlation: (abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) =>
    load_correlation(abort_controller, kyous, query, kyou_is_loaded),
  reset,
})
</script>

<style scoped>
.dnote_correlation_graph {
  padding: 0 4px;
}

.dnote_correlation_graph h2 {
  white-space: nowrap;
}

.dnote_correlation_graph h2.draggable {
  cursor: grab;
  user-select: none;
}

/* 指標が増えると行列が画面幅を超えるので、行列だけを横スクロールさせる */
.correlation_matrix_scroll {
  overflow-x: auto;
}

.correlation_matrix {
  display: grid;
  min-width: max-content;
  gap: 2px;
  align-items: stretch;
}

.matrix_corner,
.matrix_header,
.matrix_cell {
  min-height: 52px;
  padding: 6px;
}

.matrix_header {
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  text-align: center;
}

/* 横スクロールしても指標名が読めるよう、見出しは行・列とも貼り付ける。
   交点(corner)は行・列の両方に重なるので z-index を一番上にする */
.matrix_corner {
  position: sticky;
  left: 0;
  top: 0;
  z-index: 3;
  background: rgb(var(--v-theme-surface));
}

.column_header {
  position: sticky;
  top: 0;
  z-index: 2;
  background: rgb(var(--v-theme-surface));
}

.row_header {
  justify-content: flex-start;
  position: sticky;
  left: 0;
  z-index: 1;
  background: rgb(var(--v-theme-surface));
}

.matrix_cell {
  border: 2px solid transparent;
  border-radius: 4px;
  color: rgb(var(--v-theme-on-surface));
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

/* 選択枠は背景色に埋もれることがあるため、内側にもう1本線を重ねる */
.matrix_cell.selected {
  border-color: rgb(var(--v-theme-highlight));
  box-shadow: inset 0 0 0 1px rgb(var(--v-theme-on-surface));
}

.matrix_cell small {
  font-size: 0.7rem;
}

.correlation_scatter {
  display: block;
  width: 100%;
  min-height: 240px;
  /* ダブルタップズーム由来のclick遅延を除去する（パンスクロールは維持される） */
  touch-action: manipulation;
}

.scatter_axis {
  stroke: rgb(var(--v-theme-on-surface));
  stroke-width: 1;
}

.scatter_label {
  fill: rgb(var(--v-theme-on-surface));
  font-size: 13px;
}

.scatter_point {
  fill: rgb(var(--v-theme-primary));
  stroke: rgb(var(--v-theme-on-primary));
  stroke-width: 1;
  cursor: pointer;
}

.scatter_point:focus {
  outline: none;
  stroke: rgb(var(--v-theme-highlight));
  stroke-width: 3;
}

.scatter_point_detail {
  text-align: center;
  overflow-wrap: anywhere;
}
</style>
