<template>
  <div class="dnote_correlation_graph_table_root" @dragover="onTableDragover" @drop="onTableDrop">
    <div class="dnote_correlation_graph_table_row">
      <DnoteCorrelationGraphView v-for="(q, index) in model_value" :key="q.id" v-model="model_value[index]"
        :editable="editable" :application_config="application_config" :gkill_api="gkill_api"
        @requested_move_dnote_correlation_graph="(src_id: string, target_id: string, direction: 'left' | 'right') => handle_move_dnote_correlation_graph(src_id, target_id, direction)"
        @requested_delete_dnote_correlation_graph="(id: string) => delete_dnote_correlation_graph(id)"
        @requested_update_dnote_correlation_graph="(qq: DnoteCorrelationGraphQuery) => update_dnote_correlation_graph(qq)"
        @finish_a_aggregate_task="emits('finish_a_aggregate_task')"
        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
        ref="dnote_correlation_graph_views" />
    </div>
  </div>
</template>

<script setup lang="ts">
import DnoteCorrelationGraphView from "./dnote-correlation-graph-view.vue"
import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { DnoteCorrelationGraphQuery } from "@/classes/dnote/dnote-correlation"
import type DnoteCorrelationGraphTableViewEmits from "./dnote-correlation-graph-table-view-emits"
import type DnoteCorrelationGraphTableViewProps from "./dnote-correlation-graph-table-view-props"
import { useDnoteCorrelationGraphTableView } from "@/classes/use-dnote-correlation-graph-table-view"

const props = defineProps<DnoteCorrelationGraphTableViewProps>()
const emits = defineEmits<DnoteCorrelationGraphTableViewEmits>()

const model_value = defineModel<Array<DnoteCorrelationGraphQuery>>({ default: () => [] })

const {
  // Template refs
  dnote_correlation_graph_views,

  // Methods used in template
  handle_move_dnote_correlation_graph,
  delete_dnote_correlation_graph,
  update_dnote_correlation_graph,
  onTableDragover,
  onTableDrop,

  // Exposed methods
  load_correlation,
  reset,
} = useDnoteCorrelationGraphTableView({ props, emits, model_value })

defineExpose({ load_correlation, reset })
</script>

<style scoped>
.dnote_correlation_graph_table_root {
  overflow-x: visible;
  padding: 0px;
}

.dnote_correlation_graph_table_row {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
