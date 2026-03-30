'use strict'

import { nextTick, ref } from 'vue'
import type DnoteItemListViewProps from '@/pages/views/dnote-item-list-view-props'
import type DnoteItemListViewEmits from '@/pages/views/dnote-item-list-view-emits'
import type DnoteItem from '@/classes/dnote/dnote-item'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type { Ref } from 'vue'
import type { ComponentRef } from '@/classes/component-ref'

export function useDnoteItemListView(options: {
    props: DnoteItemListViewProps
    emits: DnoteItemListViewEmits
    model_value: Ref<Array<DnoteItem>>
}) {
    const { props, emits, model_value } = options

    const dnote_item_views = ref<ComponentRef | null>(null)
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
            for (let i = 0; i < dnote_item_views.value!.length; i++) {
                await dnote_item_views.value![i].reset()
            }
        })
    }

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

    return {
        dnote_item_views,
        dnd_list_index,
        load_aggregated_value,
        delete_dnote_item,
        update_dnote_item,
        reset,
        on_list_dragover,
        on_list_drop,
    }
}
