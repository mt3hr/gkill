<template>
  <div class="dnote_trend_graph_table_root" @dragover="onTableDragover" @drop="onTableDrop">
    <div class="dnote_trend_graph_table_row">
      <DnoteTrendGraphView v-for="(q, index) in model_value" :key="q.id" v-model="model_value[index]"
        :editable="editable" :application_config="application_config" :gkill_api="gkill_api"
        @requested_move_dnote_trend_graph="(src_id: string, target_id: string, direction: 'left' | 'right') => handle_move_dnote_trend_graph(src_id, target_id, direction)"
        @requested_delete_dnote_trend_graph="(id: string) => delete_dnote_trend_graph(id)"
        @requested_update_dnote_trend_graph="(qq: DnoteTrendGraphQuery) => update_dnote_trend_graph(qq)"
        @finish_a_aggregate_task="emits('finish_a_aggregate_task')"
        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
        ref="dnote_trend_graph_views" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import DnoteTrendGraphView from "./dnote-trend-graph-view.vue"
import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type DnoteTrendGraphTableViewEmits from "./dnote-trend-graph-table-view-emits"
import type DnoteTrendGraphTableViewProps from "./dnote-trend-graph-table-view-props"
import { useDnoteTrendGraphTableView } from '@/classes/use-dnote-trend-graph-table-view'
import type DnoteTrendGraphQuery from "./dnote-trend-graph-query"

const props = defineProps<DnoteTrendGraphTableViewProps>()
const emits = defineEmits<DnoteTrendGraphTableViewEmits>()

const model_value = defineModel<Array<DnoteTrendGraphQuery>>({ default: () => [] })

const {
    // Template refs
    dnote_trend_graph_views,

    // Methods used in template
    handle_move_dnote_trend_graph,
    delete_dnote_trend_graph,
    update_dnote_trend_graph,
    onTableDragover,
    onTableDrop,

    // Exposed methods
    load_trend_graph,
    reset,
} = useDnoteTrendGraphTableView({ props, emits, model_value })

defineExpose({ load_trend_graph, reset })
</script>

<style scoped>
.dnote_trend_graph_table_root {
  overflow-x: visible;
  padding: 0px;
}

.dnote_trend_graph_table_row {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
