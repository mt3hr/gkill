<template>
  <div
    class="dnote_list_view_root"
    @dragover="dragover"
    @drop="drop"
  >
    <!-- ドラッグはタイトルをハンドルにする（スクロール操作と干渉しにくい） -->
    <h2
      class="dnote_list_title"
      :draggable="editable"
      :class="{ draggable: editable }"
      @dragstart="drag_start"
    >
      {{ model_value?.title ? model_value.title : "" }}
    </h2>

    <v-virtual-scroll
      class="dnote_list_view"
      :items="aggregated_items"
      :height="'50vh'"
      :width="200 + 8"
      @contextmenu.prevent.stop="onContextmenu"
      ref="list_view"
    >
      <template v-slot:default="{ item }">
        <AggregatedListItem
          :application_config="application_config"
          :gkill_api="gkill_api"
          :dnote_list_query="model_value!"
          :aggregated_item="item"
          v-on="aggregatedListItemHandlers"
        />
      </template>
    </v-virtual-scroll>

    <v-card variant="text" :ripple="false" :link="false">
      <v-row no-gutters>
        <v-col v-if="aggregated_items && aggregated_items.length" cols="auto" class="py-3">
          {{ aggregated_items.length }}{{ i18n.global.t("N_COUNT_ITEMS_TITLE") }}
        </v-col>
        <v-spacer />
      </v-row>
    </v-card>

    <DnoteListQueryContextMenu
      :application_config="application_config"
      :gkill_api="gkill_api"
      v-on="contextMenuHandlers"
      ref="contextmenu"
    />

    <ConfirmDeleteDnoteListQueryDialog
      :application_config="application_config"
      :gkill_api="gkill_api"
      v-on="confirmDeleteHandlers"
      ref="confirm_delete_dnote_list_query_dialog"
    />

    <EditDnoteListDialog
      :application_config="application_config"
      :gkill_api="gkill_api"
      :dnote_list_query="model_value!"
      v-on="editDnoteListHandlers"
      ref="edit_dnote_list_query"
    />
  </div>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import AggregatedListItem from './aggregated-list-item.vue'
import EditDnoteListDialog from '../dialogs/edit-dnote-list-dialog.vue'
import DnoteListQueryContextMenu from './dnote-list-query-context-menu.vue'
import ConfirmDeleteDnoteListQueryDialog from '../dialogs/confirm-delete-dnote-list-query-dialog.vue'
import type DnoteListViewProps from './dnote-list-view-props'
import type DnoteListQuery from './dnote-list-query'
import type DnoteListViewEmits from './dnote-list-view-emits'
import { useDnoteListView } from '@/classes/use-dnote-list-view'

const props = defineProps<DnoteListViewProps>()
const emits = defineEmits<DnoteListViewEmits>()
const model_value = defineModel<DnoteListQuery>()

const {
  // Template refs
  list_view,
  contextmenu,
  confirm_delete_dnote_list_query_dialog,
  edit_dnote_list_query,

  // State
  aggregated_items,

  // Business logic
  load_aggregate_grouping_list,
  reset,

  // DnD
  drag_start,
  dragover,
  drop,

  // Template event handlers
  onContextmenu,

  // Event relay objects
  aggregatedListItemHandlers,
  contextMenuHandlers,
  confirmDeleteHandlers,
  editDnoteListHandlers,
} = useDnoteListView({ props, emits, model_value })

defineExpose({ load_aggregate_grouping_list, reset })
</script>

<style scoped>
.dnote_list_title.draggable {
  cursor: grab;
  user-select: none;
}
.dnote_list_title.draggable:active {
  cursor: grabbing;
}
</style>
