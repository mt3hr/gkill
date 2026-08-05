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
        const wait_promises: Array<Promise<void>> = []
        for (let i = 0; i < dnote_item_views.value.length; i++) {
            const v = dnote_item_views.value[i]
            if (!v) continue
            wait_promises.push(v.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded))
        }
        return Promise.all(wait_promises)
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

    function onListDragover(e: DragEvent): void {
        if (!props.editable) return
        e.preventDefault()
        if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
    }

    function onListDrop(e: DragEvent): void {
        if (!props.editable) return

        const src_id = e.dataTransfer?.getData("gkill_dnote_item_id")
        const src_list_index_str = e.dataTransfer?.getData("gkill_dnote_item_src_list_index")
        if (!src_id || src_list_index_str === undefined || src_list_index_str === null || src_list_index_str === "") return

        const src_list_index = Number(src_list_index_str)
        const el = e.currentTarget as HTMLElement | null
        if (!el) return
        const rect = el.getBoundingClientRect()
        const y = e.clientY - rect.top
        const drop_type: "up" | "down" = y <= rect.height * 0.5 ? "up" : "down"

        emits("requested_move_dnote_item", src_id, src_list_index, null, dnd_list_index, drop_type)
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
        onListDragover,
        onListDrop,
    }
}
