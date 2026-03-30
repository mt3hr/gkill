<template>
  <div class="dnote_list_table_root" @dragover="on_table_dragover" @drop="on_table_drop">
    <div class="dnote_list_table_row">
      <DnoteListView v-for="(q, index) in model_value" :key="q.id" v-model="model_value[index]" :editable="editable"
        :application_config="application_config" :gkill_api="gkill_api"
        @requested_move_dnote_list_query="(list_id: string, query_id: string, direction: 'left' | 'right') => handle_move_dnote_list_query(list_id, query_id, direction)"
        @requested_delete_dnote_list_query="(id: string) => delete_dnote_list_query(id)"
        @requested_update_dnote_list_query="(qq: DnoteListQuery) => update_dnote_list_query(qq)"
        @finish_a_aggregate_task="emits('finish_a_aggregate_task')"
        @focused_kyou="(kyou: Kyou) => emits('focused_kyou', kyou)"
        @clicked_kyou="(kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) }"
        @deleted_kyou="(kyou: Kyou) => emits('deleted_kyou', kyou)"
        @deleted_tag="(tag: Tag) => emits('deleted_tag', tag)"
        @deleted_text="(text: Text) => emits('deleted_text', text)"
        @deleted_notification="(n: Notification) => emits('deleted_notification', n)"
        @registered_kyou="(kyou: Kyou) => emits('registered_kyou', kyou)"
        @registered_tag="(tag: Tag) => emits('registered_tag', tag)"
        @registered_text="(text: Text) => emits('registered_text', text)"
        @registered_notification="(n: Notification) => emits('registered_notification', n)"
        @updated_kyou="(kyou: Kyou) => emits('updated_kyou', kyou)"
        @updated_tag="(tag: Tag) => emits('updated_tag', tag)"
        @updated_text="(text: Text) => emits('updated_text', text)"
        @updated_notification="(n: Notification) => emits('updated_notification', n)"
        @requested_open_rykv_dialog="(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload)"
        ref="dnote_list_views" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import DnoteListView from "./dnote-list-view.vue"
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"
import type { Text } from "@/classes/datas/text"
import type { Notification } from "@/classes/datas/notification"
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
  padding: 0px;
}

.dnote_list_table_row {
  display: flex;
  gap: 0px;
  align-items: flex-start;
  min-height: 81px;
}
</style>
