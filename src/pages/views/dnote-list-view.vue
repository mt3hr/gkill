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
      @contextmenu.prevent.stop="(e: any) => { if (editable) { contextmenu?.show(e, model_value!.id) } }"
      ref="list_view"
    >
      <template v-slot:default="{ item }">
        <AggregatedListItem
          :application_config="application_config"
          :gkill_api="gkill_api"
          :dnote_list_query="model_value!"
          :aggregated_item="item"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_delete_dnote_list_query="(...id: any[]) => emits('requested_delete_dnote_list_query', id[0] as string)"
          @requested_update_dnote_list_query="(...dnote_list_query: any[]) => emits('requested_update_dnote_list_query', dnote_list_query[0] as DnoteListQuery)"
          @deleted_kyou="(...kyou: any[]) => emits('deleted_kyou', kyou[0] as Kyou)"
          @deleted_tag="(...tag: any[]) => emits('deleted_tag', tag[0] as Tag)"
          @deleted_text="(...text: any[]) => emits('deleted_text', text[0] as Text)"
          @deleted_notification="(...notification: any[]) => emits('deleted_notification', notification[0] as Notification)"
          @registered_kyou="(...kyou: any[]) => emits('registered_kyou', kyou[0] as Kyou)"
          @registered_tag="(...tag: any[]) => emits('registered_tag', tag[0] as Tag)"
          @registered_text="(...text: any[]) => emits('registered_text', text[0] as Text)"
          @registered_notification="(...notification: any[]) => emits('registered_notification', notification[0] as Notification)"
          @updated_kyou="(...kyou: any[]) => emits('updated_kyou', kyou[0] as Kyou)"
          @updated_tag="(...tag: any[]) => emits('updated_tag', tag[0] as Tag)"
          @updated_text="(...text: any[]) => emits('updated_text', text[0] as Text)"
          @updated_notification="(...notification: any[]) => emits('updated_notification', notification[0] as Notification)"
          @requested_open_rykv_dialog="(...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2])"
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
      @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
      @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
      @requested_delete_dnote_list_query="confirm_delete_dnote_list_query_dialog?.show(model_value!)"
      @requested_edit_dnote_list_query="edit_dnote_list_query?.show()"
      ref="contextmenu"
    />

    <ConfirmDeleteDnoteListQueryDialog
      :application_config="application_config"
      :gkill_api="gkill_api"
      @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
      @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
      @requested_delete_dnote_list_query="(...id: any[]) => emits('requested_delete_dnote_list_query', id[0] as string)"
      ref="confirm_delete_dnote_list_query_dialog"
    />

    <EditDnoteListDialog
      :application_config="application_config"
      :gkill_api="gkill_api"
      :dnote_list_query="model_value!"
      @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
      @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
      @requested_update_dnote_list_query="(...dnote_list_query: any[]) => emits('requested_update_dnote_list_query', dnote_list_query[0] as DnoteListQuery)"
      ref="edit_dnote_list_query"
    />
  </div>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref } from 'vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type { Text } from '@/classes/datas/text';
import type { Tag } from '@/classes/datas/tag';
import type { Notification } from '@/classes/datas/notification';
import type AgregatedItem from '../../classes/dnote/aggregate-grouping-list-result-record';
import AggregatedListItem from './aggregated-list-item.vue';
import { DnoteListAggregator } from '../../classes/dnote/dnote-list-aggregator';
import type DnoteListViewProps from './dnote-list-view-props';
import type DnoteListQuery from './dnote-list-query';
import type DnoteListViewEmits from './dnote-list-view-emits';
import EditDnoteListDialog from '../dialogs/edit-dnote-list-dialog.vue';
import DnoteListQueryContextMenu from './dnote-list-query-context-menu.vue';
import ConfirmDeleteDnoteListQueryDialog from '../dialogs/confirm-delete-dnote-list-query-dialog.vue';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

const props = defineProps<DnoteListViewProps>()
defineExpose({ load_aggregate_grouping_list, reset })
const emits = defineEmits<DnoteListViewEmits>()
const model_value = defineModel<DnoteListQuery>()

const contextmenu = ref<InstanceType<typeof DnoteListQueryContextMenu> | null>(null);
const confirm_delete_dnote_list_query_dialog = ref<InstanceType<typeof ConfirmDeleteDnoteListQueryDialog> | null>(null);
const edit_dnote_list_query = ref<InstanceType<typeof EditDnoteListDialog> | null>(null);

const aggregated_items: Ref<Array<AgregatedItem>> = ref(new Array<AgregatedItem>())

async function load_aggregate_grouping_list(
  abort_controller: AbortController,
  kyous: Array<Kyou>,
  find_kyou_query: FindKyouQuery,
  kyou_is_loaded: boolean
): Promise<void> {
  if (!model_value.value) return

  const list_aggregator = new DnoteListAggregator(
    model_value.value.predicate,
    model_value.value.key_getter,
    model_value.value.aggregate_target
  )
  const aggregated_result = await list_aggregator.aggregate_grouping_list(
    abort_controller,
    kyous,
    find_kyou_query,
    kyou_is_loaded
  )

  aggregated_items.value.splice(0)
  for (let i = 0; i < aggregated_result.length; i++) {
    aggregated_items.value.push(aggregated_result[i])
  }
  emits('finish_a_aggregate_task')
}

async function reset(): Promise<void> {
  return nextTick(async () => {
    aggregated_items.value.splice(0)
  })
}

/**
 * DnD（列移動 / FoldableStruct式の水平版）
 * 左半分: left（前に挿入）
 * 右半分: right（後に挿入）
 */
type DropType = 'left' | 'right'

function drag_start(e: DragEvent): void {
  if (!props.editable) return
  const id = model_value.value?.id ?? ''
  if (!id) return

  e.dataTransfer?.setData('gkill_dnote_list_id', id)
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
  e.stopPropagation()
}

function dragover(e: DragEvent): void {
  if (!props.editable) return
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  e.preventDefault() // drop許可
}

function drop(e: DragEvent): void {
  if (!props.editable) return

  const srcId = e.dataTransfer?.getData('gkill_dnote_list_id')
  const targetId = model_value.value?.id ?? ''
  if (!srcId || !targetId) return
  if (srcId === targetId) return

  const el = e.currentTarget as HTMLElement | null
  if (!el) return

  const rect = el.getBoundingClientRect()
  const x = e.clientX - rect.left
  const dropType: DropType = (x <= rect.width * 0.5) ? 'left' : 'right'

  emits('requested_move_dnote_list_query', srcId, targetId, dropType)

  e.preventDefault()
  e.stopPropagation()
}
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
