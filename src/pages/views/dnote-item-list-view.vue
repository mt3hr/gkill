<template>
    <div>
        <DnoteItemView v-for="dnote_item in model_value" v-model="dnote_item!" :key="dnote_item.id" :editable="editable"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_item="(id) => delete_dnote_item(id)"
            @requested_update_dnote_item="(dnote_item) => update_dnote_item(dnote_item)"
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
            @finish_a_aggregate_task="emits('finish_a_aggregate_task')" ref="dnote_item_views" />
    </div>
</template>

<script lang="ts" setup>
import { nextTick, ref } from 'vue';
import type DnoteItemListViewProps from './dnote-item-list-view-props';
import DnoteItemView from './dnote-item-view.vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type DnoteItem from '../../classes/dnote/dnote-item';
import type DnoteItemListViewEmits from './dnote-item-list-view-emits';

const dnote_item_views = ref()

defineProps<DnoteItemListViewProps>()
defineExpose({ load_aggregated_value, reset })
const emits = defineEmits<DnoteItemListViewEmits>()
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

async function reset(): Promise<void> {
    if (!dnote_item_views.value || dnote_item_views.value.length === 0) {
        return
    }
    return nextTick(async () => {
        for (let i = 0; i < dnote_item_views.value.length; i++) {
            await dnote_item_views.value[i].reset()
        }
    })
}

</script>