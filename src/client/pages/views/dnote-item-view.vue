<template>
  <div class="dnote_item_root" :draggable="editable" :class="{ draggable: editable }" @dragstart="drag_start"
    @dragover="dragover" @drop="drop"
    @contextmenu.prevent.stop="onContextmenu"
    @dblclick="onDblclick">
    <table>
      <tr>
        <td>
          <span class="title"><span>{{ model_value!.title }}</span><span>:</span></span>
        </td>
        <td><span>{{ model_value!.prefix }}</span></td>
        <td>
          <span class="value">
            <span v-if="!is_lantana_type" :class="value_class">{{ value }}</span>
            <span v-else>
              <LantanaFlowersView :gkill_api="gkill_api" :application_config="application_config" :mood="mood_value"
                :editable="false" />
            </span>
          </span>
        </td>
        <td><span>{{ model_value!.suffix }}</span></td>
      </tr>
    </table>

    <KyouListViewDialog v-model="related_kyous" :kyou_height="180" :width="400" :application_config="application_config"
      :gkill_api="gkill_api" :is_focused_list="true" :closable="false" :highlight_targets="[]"
      :list_height="list_height" :enable_context_menu="true" :enable_dialog="true" :is_readonly_mi_check="true"
      :show_checkbox="true" :show_footer="false" :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true"
      :show_content_only="false" :show_rep_name="true" :force_show_latest_kyou_info="true"
      :show_timeis_plaing_end_button="false"
      v-on="kyouListViewDialogHandlers"
      ref="kyou_list_view_dialog" />

    <DnoteItemListContextMenu :application_config="application_config" :gkill_api="gkill_api"
      v-on="contextMenuHandlers"
      ref="contextmenu" />

    <ConfirmDeleteDnoteItemListDialog :application_config="application_config" :gkill_api="gkill_api"
      v-on="confirmDeleteHandlers"
      ref="confirm_delete_dnote_item_list_dialog" />

    <EditDnoteItemDialog :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
      v-on="editDnoteItemHandlers"
      ref="edit_dnote_item_dialog" />
  </div>
</template>

<script lang="ts" setup>
import type DnoteItemProps from "./dnote-item-props"
import type DnoteItemViewEmits from "./dnote-item-view-emits"
import type DnoteItem from "@/classes/dnote/dnote-item"
import LantanaFlowersView from "./lantana-flowers-view.vue"
import DnoteItemListContextMenu from "./dnote-item-list-context-menu.vue"
import ConfirmDeleteDnoteItemListDialog from "../dialogs/confirm-delete-dnote-item-list-dialog.vue"
import EditDnoteItemDialog from "../dialogs/edit-dnote-item-dialog.vue"
import KyouListViewDialog from "../dialogs/kyou-list-view-dialog.vue"
import { useDnoteItemView } from "@/classes/use-dnote-item-view"

const props = defineProps<DnoteItemProps>()
const emits = defineEmits<DnoteItemViewEmits>()
const model_value = defineModel<DnoteItem>()

const {
  // Template refs
  contextmenu,
  confirm_delete_dnote_item_list_dialog,
  edit_dnote_item_dialog,
  kyou_list_view_dialog,

  // State
  value,
  related_kyous,
  list_height,

  // Computed
  is_lantana_type,
  value_class,
  mood_value,

  // Business logic
  load_aggregated_value,
  reset,

  // DnD
  drag_start,
  dragover,
  drop,

  // Template event handlers
  onContextmenu,
  onDblclick,

  // Event relay objects
  kyouListViewDialogHandlers,
  contextMenuHandlers,
  confirmDeleteHandlers,
  editDnoteItemHandlers,
} = useDnoteItemView({ props, emits, model_value })

defineExpose({ load_aggregated_value, reset })
</script>

<style scoped>
.dnote_item_root.draggable {
  cursor: grab;
}
</style>
