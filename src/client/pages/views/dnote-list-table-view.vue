<template>
  <div class="dnote_list_table_root" @dragover="on_table_dragover" @drop="on_table_drop">
    <div class="dnote_list_table_row">
      <DnoteListView v-for="(q, index) in model_value" :key="q.id" v-model="model_value[index]" :editable="editable"
        :application_config="application_config" :gkill_api="gkill_api"
        @requested_move_dnote_list_query="(...args: any[]) => handle_move_dnote_list_query(args[0] as string, args[1] as string, args[2] as 'left' | 'right')"
        @requested_delete_dnote_list_query="(...id: any[]) => delete_dnote_list_query(id[0] as string)"
        @requested_update_dnote_list_query="(...qq: any[]) => update_dnote_list_query(qq[0])"
        @finish_a_aggregate_task="emits('finish_a_aggregate_task')"
        @focused_kyou="(...kyou: any[]) => emits('focused_kyou', kyou[0])"
        @clicked_kyou="(...kyou: any[]) => { emits('focused_kyou', kyou[0]); emits('clicked_kyou', kyou[0]) }"
        @deleted_kyou="(...kyou: any[]) => emits('deleted_kyou', kyou[0])"
        @deleted_tag="(...tag: any[]) => emits('deleted_tag', tag[0])"
        @deleted_text="(...text: any[]) => emits('deleted_text', text[0])"
        @deleted_notification="(...n: any[]) => emits('deleted_notification', n[0])"
        @registered_kyou="(...kyou: any[]) => emits('registered_kyou', kyou[0])"
        @registered_tag="(...tag: any[]) => emits('registered_tag', tag[0])"
        @registered_text="(...text: any[]) => emits('registered_text', text[0])"
        @registered_notification="(...n: any[]) => emits('registered_notification', n[0])"
        @updated_kyou="(...kyou: any[]) => emits('updated_kyou', kyou[0])"
        @updated_tag="(...tag: any[]) => emits('updated_tag', tag[0])"
        @updated_text="(...text: any[]) => emits('updated_text', text[0])"
        @updated_notification="(...n: any[]) => emits('updated_notification', n[0])"
        @requested_open_rykv_dialog="(...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2])"
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
    on_table_dragover,
    on_table_drop,

    // Exposed methods
    load_aggregate_grouping_list,
    reset,
} = useDnoteListTableView({ props, emits, model_value })

defineExpose({ load_aggregate_grouping_list, reset })
</script>

<style scoped>
.dnote_list_table_root {
  overflow-x: auto;
  padding: 4px;
}

.dnote_list_table_row {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  min-height: 81px;
}
</style>
