<template>
    <div>
        <DnoteItemListView v-for="dnote_items in model_value"
            :key="(dnote_items && dnote_items.length > 0) ? dnote_items[0].id : 0" v-model="dnote_items!"
            :editable="editable" :application_config="application_config" :gkill_api="gkill_api"
            @deleted_kyou="(kyou) => emits('deleted_kyou', kyou)" @deleted_tag="(tag) => emits('deleted_tag', tag)"
            @deleted_text="(text) => emits('deleted_text', text)"
            @deleted_notification="(notification) => emits('deleted_notification', notification)"
            @registered_kyou="(kyou) => emits('registered_kyou', kyou)"
            @registered_tag="(tag) => emits('registered_tag', tag)"
            @registered_text="(text) => emits('registered_text', text)"
            @registered_notification="(notification) => emits('registered_notification', notification)"
            @updated_kyou="(kyou) => emits('updated_kyou', kyou)" @updated_tag="(tag) => emits('updated_tag', tag)"
            @updated_text="(text) => emits('updated_text', text)"
            @updated_notification="(notification) => emits('updated_notification', notification)"
            ref="dnote_item_list_views" />
    </div>
</template>

<script lang="ts" setup>
import DnoteItemListView from './dnote-item-list-view.vue';
import type DnoteItemTableProps from './dnote-item-table-props';
import { nextTick, ref } from 'vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type DnoteItem from '../../classes/dnote/dnote-item';
import { type KyouViewEmits } from './kyou-view-emits';

const dnote_item_list_views = ref()

const model_value = defineModel<Array<Array<DnoteItem>>>()
defineProps<DnoteItemTableProps>()
defineExpose({ load_aggregated_value, reset })
const emits = defineEmits<KyouViewEmits>()

async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean): Promise<any> {
    if (!dnote_item_list_views.value || !model_value.value) {
        return
    }
    const waitPromises = new Array<Promise<void>>()
    for (let i = 0; i < dnote_item_list_views.value.length; i++) {
        if (!(dnote_item_list_views.value as any)[i]) {
            continue
        }
        const dnote_item_list_view = (dnote_item_list_views.value as any)[i]
        waitPromises.push(dnote_item_list_view.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded))
    }
    return await Promise.all(waitPromises)
}

async function reset(): Promise<void> {
    if (!dnote_item_list_views.value || dnote_item_list_views.value.length === 0) {
        return
    }
    return nextTick(async () => {
        for (let i = 0; i < dnote_item_list_views.value.length; i++) {
            await dnote_item_list_views.value[i].reset()
        }
    })
}
</script>