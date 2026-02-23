<template>
    <div class="dnote_item_list_root" @dragover="on_list_dragover" @drop="on_list_drop">
        <DnoteItemView v-for="(dnote_item, index) in model_value" :key="dnote_item.id" v-model="model_value[index]"
            :editable="editable" :dnd_list_index="dnd_list_index" :application_config="application_config"
            :gkill_api="gkill_api"
            @requested_move_dnote_item="(...args: any[]) => emits('requested_move_dnote_item', args[0] as string, args[1] as number, args[2] as string, args[3] as number, args[4] as 'up' | 'down')"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0])"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0])"
            @focused_kyou="(...kyou: any[]) => emits('focused_kyou', kyou[0])"
            @clicked_kyou="(...kyou: any[]) => { emits('focused_kyou', kyou[0]); emits('clicked_kyou', kyou[0]) }"
            @requested_delete_dnote_item="(...id: any[]) => delete_dnote_item(id[0] as string)"
            @requested_update_dnote_item="(...d: any[]) => update_dnote_item(d[0])"
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
            @finish_a_aggregate_task="emits('finish_a_aggregate_task')" ref="dnote_item_views" />
    </div>
</template>

<script lang="ts" setup>
import { nextTick, ref } from "vue"
import DnoteItemView from "./dnote-item-view.vue"
import type DnoteItemListViewProps from "./dnote-item-list-view-props"
import type DnoteItemListViewEmits from "./dnote-item-list-view-emits"
import type DnoteItem from "../../classes/dnote/dnote-item"
import type { FindKyouQuery } from "../../classes/api/find_query/find-kyou-query"
import type { Kyou } from "../../classes/datas/kyou"

const props = defineProps<DnoteItemListViewProps>()
const emits = defineEmits<DnoteItemListViewEmits>()
defineExpose({ load_aggregated_value, reset })

const model_value = defineModel<Array<DnoteItem>>({ default: () => [] })
const dnote_item_views = ref<any>(null)

// ★ snake_case を採用
const dnd_list_index = props.dnd_list_index

async function load_aggregated_value(
    abort_controller: AbortController,
    kyous: Array<Kyou>,
    query: FindKyouQuery,
    kyou_is_loaded: boolean
) {
    if (!dnote_item_views.value) return
    const waitPromises: Array<Promise<void>> = []
    for (let i = 0; i < dnote_item_views.value.length; i++) {
        const v = dnote_item_views.value[i]
        if (!v) continue
        waitPromises.push(v.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded))
    }
    return Promise.all(waitPromises)
}

function delete_dnote_item(dnote_item_id: string): void {
    const idx = model_value.value.findIndex((x) => x.id === dnote_item_id)
    if (idx < 0) return
    model_value.value.splice(idx, 1)
}

function update_dnote_item(dnote_item: DnoteItem): void {
    const idx = model_value.value.findIndex((x) => x.id === dnote_item.id)
    if (idx < 0) return
    model_value.value.splice(idx, 1, dnote_item)
}

async function reset(): Promise<void> {
    if (!dnote_item_views.value || dnote_item_views.value.length === 0) return
    return nextTick(async () => {
        for (let i = 0; i < dnote_item_views.value.length; i++) {
            await dnote_item_views.value[i].reset()
        }
    })
}

/** 列の空白領域への drop（先頭/末尾） */
function on_list_dragover(e: DragEvent): void {
    if (!props.editable) return
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
}

function on_list_drop(e: DragEvent): void {
    if (!props.editable) return

    const srcId = e.dataTransfer?.getData("gkill_dnote_item_id")
    const srcListIndexStr = e.dataTransfer?.getData("gkill_dnote_item_src_list_index")
    if (!srcId || srcListIndexStr === undefined || srcListIndexStr === null || srcListIndexStr === "") return

    const srcListIndex = Number(srcListIndexStr)
    const el = e.currentTarget as HTMLElement | null
    if (!el) return
    const rect = el.getBoundingClientRect()
    const y = e.clientY - rect.top
    const dropType: "up" | "down" = y <= rect.height * 0.5 ? "up" : "down"

    emits("requested_move_dnote_item", srcId, srcListIndex, null, dnd_list_index, dropType)
    e.preventDefault()
    e.stopPropagation()
}
</script>

<style scoped>
.dnote_item_list_root {
    min-height: 40px;
}
</style>
