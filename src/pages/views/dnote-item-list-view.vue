<template>
    <div>
        <DnoteItemView v-for="dnote_item in model_value" v-model="dnote_item!" :key="dnote_item.id" :editable="editable"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_item="(id) => delete_dnote_item(id)"
            @requested_update_dnote_item="(dnote_item) => update_dnote_item(dnote_item)" ref="dnote_item_views" />
    </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import type DnoteItemListViewProps from './dnote-item-list-view-props';
import DnoteItemView from './dnote-item-view.vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type DnoteItem from '../../classes/dnote/dnote-item';
import type { KyouViewEmits } from './kyou-view-emits';

const dnote_item_views = ref()

defineProps<DnoteItemListViewProps>()
defineExpose({ load_aggregated_value })
const emits = defineEmits<KyouViewEmits>()
const model_value = defineModel<Array<DnoteItem>>()

async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
    if (!dnote_item_views.value || !model_value.value) {
        return
    }
    const waitPromises = new Array<Promise<void>>()
    for (let i = 0; i < dnote_item_views.value.length; i++) {
        if (!(dnote_item_views.value as any)[i]) {
            continue
        }
        const dnote_item_view = (dnote_item_views.value as any)[i]
        waitPromises.push(dnote_item_view.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded))
    }
    return Promise.all(waitPromises)
}

function delete_dnote_item(dnote_item_id: string): void {
    let delete_target_index: number | null = null;
    for (let i = 0; i < model_value.value!.length; i++) {
        const dnote_item = model_value.value![i]
        if (dnote_item.id === dnote_item_id) {
            delete_target_index = i
            break
        }
    }
    if (delete_target_index === null) {
        return
    }
    model_value.value?.splice(delete_target_index, 1)
}

function update_dnote_item(_dnote_item: DnoteItem): void {
    let delete_target_index: number | null = null;
    for (let i = 0; i < model_value.value!.length; i++) {
        const dnote_item = model_value.value![i]
        if (dnote_item.id === _dnote_item.id) {
            delete_target_index = i
            break
        }
    }
    if (delete_target_index === null) {
        return
    }
    model_value.value?.splice(delete_target_index, 1, _dnote_item)
}

</script>