<template>
  <div class="dnote_trend_graph_view_root" @dragover="dragover" @drop="drop" @click="onGraphClick"
    @touchstart="onGraphTouchstart" @touchend="onGraphTouchend" @contextmenu.prevent.stop="onContextmenu">
    <!-- ドラッグはタイトルをハンドルにする（スクロール操作と干渉しにくい） -->
    <h2 class="dnote_trend_graph_title" :draggable="editable" :class="{ draggable: editable }"
      @dragstart="drag_start">
      {{ model_value?.title ? model_value.title : "" }}
    </h2>

    <v-progress-linear v-if="is_loading" indeterminate color="primary" />
    <!-- width/heightはviewBox値。CSSで幅100%表示され、アスペクト比8:1が保たれる -->
    <v-sparkline v-else :type="sparkline_type" :model-value="sparkline_values" :labels="sparkline_labels"
      :min="sparkline_min" width="800" height="100" line-width="2" padding="16" smooth smooth-mode="monotone"
      auto-draw="once" color="primary" label-size="10" show-labels interactive :tooltip="sparkline_tooltip" />

    <div v-if="!is_loading && is_all_empty" class="text-center text-grey">
      {{ i18n.global.t('NO_RESULTS_MESSAGE') }}
    </div>

    <DnoteTrendGraphContextMenu :application_config="application_config" :gkill_api="gkill_api"
      v-on="contextMenuHandlers" ref="contextmenu" />

    <ConfirmDeleteDnoteTrendGraphDialog :application_config="application_config" :gkill_api="gkill_api"
      v-on="confirmDeleteHandlers" ref="confirm_delete_dnote_trend_graph_dialog" />

    <EditDnoteTrendGraphDialog :application_config="application_config" :gkill_api="gkill_api"
      :dnote_trend_graph_query="model_value!" v-on="editDnoteTrendGraphHandlers"
      ref="edit_dnote_trend_graph_dialog" />
  </div>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import DnoteTrendGraphContextMenu from './dnote-trend-graph-context-menu.vue'
import EditDnoteTrendGraphDialog from '../dialogs/edit-dnote-trend-graph-dialog.vue'
import ConfirmDeleteDnoteTrendGraphDialog from '../dialogs/confirm-delete-dnote-trend-graph-dialog.vue'
import type DnoteTrendGraphViewProps from './dnote-trend-graph-view-props'
import type DnoteTrendGraphQuery from './dnote-trend-graph-query'
import type DnoteTrendGraphViewEmits from './dnote-trend-graph-view-emits'
import { useDnoteTrendGraphView } from '@/classes/use-dnote-trend-graph-view'

const props = defineProps<DnoteTrendGraphViewProps>()
const emits = defineEmits<DnoteTrendGraphViewEmits>()
const model_value = defineModel<DnoteTrendGraphQuery>()

const {
  // Template refs
  contextmenu,
  confirm_delete_dnote_trend_graph_dialog,
  edit_dnote_trend_graph_dialog,

  // State
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
  onGraphTouchstart,
  onGraphTouchend,
  onGraphClick,
  onContextmenu,

  // Event relay objects
  contextMenuHandlers,
  confirmDeleteHandlers,
  editDnoteTrendGraphHandlers,
} = useDnoteTrendGraphView({ props, emits, model_value })

defineExpose({ load_trend_graph, reset })
</script>

<style scoped>
.dnote_trend_graph_view_root {
  width: 100%;
  padding: 0 4px;
  /* ダブルタップズーム由来のclick遅延を除去する（パンスクロールは維持される） */
  touch-action: manipulation;
}

/* 端のラベルがSVG境界で切れないようにする */
.dnote_trend_graph_view_root :deep(svg) {
  overflow: visible;
}

.dnote_trend_graph_title {
  white-space: nowrap;
}

.dnote_trend_graph_title.draggable {
  cursor: grab;
  user-select: none;
}

.dnote_trend_graph_title.draggable:active {
  cursor: grabbing;
}
</style>
