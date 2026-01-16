<template>
    <div class="dnote_item_table_root">
        <table class="dnote_item_table">
            <tr>
                <td v-for="(list, listIndex) in model_value" :key="listIndex" class="dnote_item_table_td"
                    @dragover="on_cell_dragover" @drop="(e) => on_cell_drop(e, listIndex)">
                    <DnoteItemListView v-model="model_value[listIndex]" :dnd_list_index="listIndex" :editable="editable"
                        :application_config="application_config" :gkill_api="gkill_api"
                        @requested_move_dnote_item="(...args: any[]) => handle_move_dnote_item(args[0] as string, args[1] as number, args[2] as string, args[3] as number, args[4] as 'up' | 'down')"
                        @finish_a_aggregate_task="emits('finish_a_aggregate_task')"
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
                        @updated_notification="(...n: any[]) => emits('updated_notification', n[0])" />
                </td>
            </tr>
        </table>
    </div>
</template>

<script lang="ts" setup>
import { nextTick, ref } from "vue"
import DnoteItemListView from "./dnote-item-list-view.vue"
import type DnoteItem from "@/classes/dnote/dnote-item"
import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { Kyou } from "@/classes/datas/kyou"
import type DnoteItemTableViewEmits from "./dnote-item-table-view-emits"
import type DnoteItemTableViewProps from "./dnote-item-table-view-props"

const props = defineProps<DnoteItemTableViewProps>()
const emits = defineEmits<DnoteItemTableViewEmits>()
defineExpose({ load_aggregated_value, reset })

const model_value = defineModel<Array<Array<DnoteItem>>>({ default: () => [] })
const dnote_item_list_views = ref<any>(null)

async function load_aggregated_value(
    abort_controller: AbortController,
    kyous: Array<Kyou>,
    query: FindKyouQuery,
    kyou_is_loaded: boolean
) {
    if (!dnote_item_list_views.value) return
    const waitPromises: Array<Promise<any>> = []
    for (let i = 0; i < dnote_item_list_views.value.length; i++) {
        const v = dnote_item_list_views.value[i]
        if (!v) continue
        waitPromises.push(v.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded))
    }
    await Promise.all(waitPromises)
}

async function reset(): Promise<void> {
    if (!dnote_item_list_views.value || dnote_item_list_views.value.length === 0) return
    return nextTick(async () => {
        for (let i = 0; i < dnote_item_list_views.value.length; i++) {
            await dnote_item_list_views.value[i].reset()
        }
    })
}

function handle_move_dnote_item(
    srcId: string,
    srcListIndex: number,
    targetId: string | null,
    targetListIndex: number,
    dropType: "up" | "down"
): void {
    if (!props.editable) return

    const srcList = model_value.value[srcListIndex]
    const targetList = model_value.value[targetListIndex]
    if (!srcList || !targetList) return

    const srcPos = srcList.findIndex((x) => x.id === srcId)
    if (srcPos < 0) return
    const [moved] = srcList.splice(srcPos, 1)

    let insertPos = 0
    if (targetId) {
        const targetPos = targetList.findIndex((x) => x.id === targetId)
        insertPos = targetPos < 0 ? (dropType === "up" ? 0 : targetList.length) : (dropType === "up" ? targetPos : targetPos + 1)
    } else {
        insertPos = dropType === "up" ? 0 : targetList.length
    }

    if (srcListIndex === targetListIndex && srcPos < insertPos) insertPos -= 1
    if (insertPos < 0) insertPos = 0
    if (insertPos > targetList.length) insertPos = targetList.length
    targetList.splice(insertPos, 0, moved)
}

function on_cell_dragover(e: DragEvent): void {
    if (!props.editable) return
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
}

function on_cell_drop(e: DragEvent, targetListIndex: number): void {
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

    handle_move_dnote_item(srcId, srcListIndex, null, targetListIndex, dropType)
    e.preventDefault()
    e.stopPropagation()
}
</script>

<style scoped>
.dnote_item_table_td {
    vertical-align: top;
    min-width: 210px;
    padding: 4px;
}
</style>
