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
import { nextTick, ref } from "vue"
import DnoteListView from "./dnote-list-view.vue"
import type DnoteListQuery from "./dnote-list-query"
import type { FindKyouQuery } from "../../classes/api/find_query/find-kyou-query"
import type { Kyou } from "../../classes/datas/kyou"
import type { KyouViewEmits } from "./kyou-view-emits"
import type { GkillAPI } from "../../classes/api/gkill-api"
import type { ApplicationConfig } from "../../classes/datas/config/application-config"

interface Props { gkill_api: GkillAPI; application_config: ApplicationConfig; editable: boolean }
interface Emits extends KyouViewEmits { (e: "finish_a_aggregate_task"): void }

const props = defineProps<Props>()
const emits = defineEmits<Emits>()
defineExpose({ load_aggregate_grouping_list, reset })

const model_value = defineModel<Array<DnoteListQuery>>({ default: () => [] })
const dnote_list_views = ref<any>(null)

function load_aggregate_grouping_list(
  abort_controller: AbortController,
  kyous: Array<Kyou>,
  query: FindKyouQuery,
  kyou_is_loaded: boolean
): void {
  if (!dnote_list_views.value) return
  const waits: Array<Promise<any>> = []
  for (let i = 0; i < dnote_list_views.value.length; i++) {
    const v = dnote_list_views.value[i]
    if (!v) continue
    waits.push(v.load_aggregate_grouping_list(abort_controller, kyous, query, kyou_is_loaded))
  }
  // fire-and-forget（呼び出し側は void を期待）
  Promise.all(waits).then(() => { })
}

async function reset(): Promise<void> {
  if (!dnote_list_views.value || dnote_list_views.value.length === 0) return
  return nextTick(async () => {
    for (let i = 0; i < dnote_list_views.value.length; i++) await dnote_list_views.value[i].reset()
  })
}

function delete_dnote_list_query(id: string): void {
  const idx = model_value.value.findIndex((x) => x.id === id)
  if (idx < 0) return
  model_value.value.splice(idx, 1)
}

function update_dnote_list_query(q: DnoteListQuery): void {
  const idx = model_value.value.findIndex((x) => x.id === q.id)
  if (idx < 0) return
  model_value.value.splice(idx, 1, q)
}

function handle_move_dnote_list_query(srcId: string, targetId: string, dropType: "left" | "right"): void {
  if (!props.editable) return
  const srcIndex = model_value.value.findIndex((x) => x.id === srcId)
  if (srcIndex < 0) return
  const [moved] = model_value.value.splice(srcIndex, 1)

  const targetIndex = model_value.value.findIndex((x) => x.id === targetId)
  if (targetIndex < 0) { model_value.value.push(moved); return }

  let insertIndex = dropType === "left" ? targetIndex : targetIndex + 1
  if (srcIndex < insertIndex) insertIndex -= 1
  if (insertIndex < 0) insertIndex = 0
  if (insertIndex > model_value.value.length) insertIndex = model_value.value.length
  model_value.value.splice(insertIndex, 0, moved)
}

function on_table_dragover(e: DragEvent): void {
  if (!props.editable) return
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
}

function on_table_drop(e: DragEvent): void {
  if (!props.editable) return
  const srcId = e.dataTransfer?.getData("gkill_dnote_list_id")
  if (!srcId) return

  const srcIndex = model_value.value.findIndex((x) => x.id === srcId)
  if (srcIndex < 0) return
  const [moved] = model_value.value.splice(srcIndex, 1)

  const el = e.currentTarget as HTMLElement | null
  if (!el) return
  const rect = el.getBoundingClientRect()
  const x = e.clientX - rect.left
  const insertIndex = x <= rect.width * 0.5 ? 0 : model_value.value.length
  model_value.value.splice(insertIndex, 0, moved)

  e.preventDefault()
  e.stopPropagation()
}
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
