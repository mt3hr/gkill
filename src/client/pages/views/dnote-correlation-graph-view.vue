<template>
  <section class="dnote_correlation_graph" @dragover="dragover" @drop="drop" @contextmenu.prevent.stop="onContextmenu">
    <!-- ドラッグはタイトルをハンドルにする（スクロール操作と干渉しにくい） -->
    <h2 :draggable="effective_draggable" :class="{ draggable: effective_draggable }" @dragstart="drag_start"
      @dblclick="onRequestedEditDnoteCorrelationGraph">{{ model_value?.title }}</h2>
    <v-progress-linear v-if="is_loading" indeterminate color="primary" />
    <template v-else-if="result">
      <!-- 指標を行と列に並べた総当たりのヒートマップ。セルを押すと下の散布図が切り替わる -->
      <div class="correlation_matrix_scroll">
        <div class="correlation_matrix" :style="matrix_style" role="grid">
          <div class="matrix_corner" />
          <!-- 見出しは狭い列の中で折り返す。span で包むのは、flex の中央寄せを保ったまま
               span 側だけに行数のクランプ箱を持たせるため -->
          <div v-for="metric in metrics" :key="`column-${metric.id}`" class="matrix_header column_header"
            :title="metric.title"><span>{{ metric.title }}</span></div>
          <template v-for="(row_metric, row_index) in metrics" :key="`row-${row_metric.id}`">
            <div class="matrix_header row_header" :title="row_metric.title"><span>{{ row_metric.title }}</span></div>
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

      <v-card v-if="selected_cell && selected_row_metric && selected_column_metric" variant="outlined"
        class="pa-1 mt-1 correlation_detail">
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
import { to_single_line } from "@/classes/format-date-time"
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
    // DnD
    effective_draggable,
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

// 時間の集計は format_duration が改行入りの文字列を返す。
// ここは SVG の <title>（ネイティブのツールチップ）と1行の <p> なので畳んでから並べる
function point_description(point: DnoteCorrelationPairPoint): string {
  const x = to_single_line(point.x_value_string || point.x.toString())
  const y = to_single_line(point.y_value_string || point.y.toString())
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
  padding: 0 2px;
}

/* nowrap だけだと長いタイトルがそれ自体で min-content の下限になり、
   rykv の td 列を広げてしまう。はみ出しは省略記号で切る */
.dnote_correlation_graph h2 {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 0.9rem;
  font-weight: 600;
  line-height: 1.3;
}

.dnote_correlation_graph h2.draggable {
  cursor: grab;
  user-select: none;
}

/* 行列は折れ線・棒グラフと同じく列の幅いっぱいに広がる（セルのトラックが 1fr）。
   contain: inline-size が要点で、「中身は横幅の計算に関与しない」ことにして、
   行列の max-content が祖先へ伝わるのを止めている。これが無いと rykv の
   width:fit-content な Dnote 列が最長の指標名ぶん広がる（指標7個で約1400px）。
   幅が足りないときだけ行列を横スクロールさせる */
.correlation_matrix_scroll {
  overflow-x: auto;
  contain: inline-size;
}

/* min-width: max-content を付けてはいけない。付けるとグリッドが max-content 制約で測られ、
   全ての列が「行列中で最も長い見出し」の幅に揃えられて、指標7個で1400px近くになる */
.correlation_matrix {
  display: grid;
  gap: 1px;
  align-items: stretch;
}

.matrix_corner,
.matrix_header,
.matrix_cell {
  min-height: 28px;
  padding: 2px;
}

.matrix_header {
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  text-align: center;
  overflow: hidden;
}

/* 見出しは指標名を全文見せる。狭い列では折り返して縦に伸びる。
   上限を超えた分だけ切り、そのときは title 属性とセルの aria-label、
   選択時の詳細カードで全文が読める。
   max-height は line-height の整数倍にして行の途中で切れないようにしてあるので、
   どちらかを変えるときは必ず対で直すこと。
   列見出しは5行 = セル幅34pxで3文字/行なので15文字まで、
   行見出しは3行 = 64px幅で5文字/行なので15文字まで全文入る */
.matrix_header>span {
  line-height: 1.2;
  overflow: hidden;
  overflow-wrap: anywhere;
}

.column_header>span {
  font-size: 0.68rem;
  max-height: 6em;
}

.row_header>span {
  font-size: 0.74rem;
  max-height: 3.6em;
  text-align: left;
}

/* 横スクロールしても指標名が読めるよう、行見出しを左に貼り付ける。
   交点(corner)は行見出しの上に重なるので z-index を上にする。
   縦方向には貼り付けない: sticky の基準になるスクロールコンテナは
   .correlation_matrix_scroll で、これは横にしかスクロールしないため top を書いても効かない
   （縦のスクローラは祖先の .dnote-scroll-wrap 側にある） */
.matrix_corner {
  position: sticky;
  left: 0;
  z-index: 2;
  background: rgb(var(--v-theme-surface));
}

.column_header {
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
  border: 1px solid transparent;
  border-radius: 3px;
  color: rgb(var(--v-theme-on-surface));
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.matrix_cell span {
  font-size: 0.8rem;
  line-height: 1.2;
}

/* 選択枠は背景色に埋もれることがあるため、内側にもう1本線を重ねる */
.matrix_cell.selected {
  border-color: rgb(var(--v-theme-highlight));
  box-shadow: inset 0 0 0 1px rgb(var(--v-theme-on-surface));
}

/* 件数が4桁を超えても行が崩れないよう、はみ出しは省略記号で切る。
   正確な件数は aria-label と選択時の詳細カードに残る */
.matrix_cell small {
  font-size: 0.62rem;
  line-height: 1.15;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 選択セルの詳細。App.vue が余白を消しているのは h1,h2 だけなので、
   p はUA既定の上下1emマージンを持つ。4段落で約48px空費していたため潰す */
.correlation_detail :deep(h3) {
  font-size: 0.85rem;
  line-height: 1.3;
  margin: 0;
  overflow-wrap: anywhere;
}

.correlation_detail :deep(p) {
  margin: 0 0 2px;
  font-size: 0.75rem;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.correlation_scatter {
  display: block;
  width: 100%;
  /* viewBox 600x320 の比率だと、幅450px未満で min-height:240px が上下に死に余白を作っていた。
     160pxなら幅300px以上でレターボックスにならない。
     preserveAspectRatio="none" は点と文字が歪むので使わない */
  min-height: 160px;
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
