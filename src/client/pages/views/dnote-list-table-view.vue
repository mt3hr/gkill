<template>
  <div class="dnote_list_table_root" @dragover="onTableDragover" @drop="onTableDrop">
    <div class="dnote_list_table_row">
      <DnoteListView v-for="(q, index) in model_value" :key="q.id" v-model="model_value[index]" :editable="editable"
        :application_config="application_config" :gkill_api="gkill_api"
        @requested_move_dnote_list_query="(list_id: string, query_id: string, direction: 'left' | 'right') => handle_move_dnote_list_query(list_id, query_id, direction)"
        @requested_delete_dnote_list_query="(id: string) => delete_dnote_list_query(id)"
        @requested_update_dnote_list_query="(qq: DnoteListQuery) => update_dnote_list_query(qq)"
        @finish_a_aggregate_task="emits('finish_a_aggregate_task')"
        v-on="crudRelayHandlers"
        ref="dnote_list_views" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import DnoteListView from "./dnote-list-view.vue"
import type DnoteListTableViewEmits from "./dnote-list-table-view-emits"
import type DnoteListTableViewProps from "./dnote-list-table-view-props"
import { useDnoteListTableView } from '@/classes/use-dnote-list-table-view'
import type DnoteListQuery from "./dnote-list-query"

const props = defineProps<DnoteListTableViewProps>()
const emits = defineEmits<DnoteListTableViewEmits>()

const model_value = defineModel<Array<DnoteListQuery>>({ default: () => [] })

const {
    // Template refs
    dnote_list_views,

    // Methods used in template
    handle_move_dnote_list_query,
    delete_dnote_list_query,
    update_dnote_list_query,
    onTableDragover,
    onTableDrop,

    // Exposed methods
    load_aggregate_grouping_list,
    reset,

    // Event relay objects
    crudRelayHandlers,
} = useDnoteListTableView({ props, emits, model_value })

defineExpose({ load_aggregate_grouping_list, reset })
</script>

<style scoped>
.dnote_list_table_root {
  overflow-x: visible;
  padding: 0px;
}

.dnote_list_table_row {
  display: flex;
  gap: 0px;
  align-items: flex-start;
  min-height: 81px;
}
</style>
