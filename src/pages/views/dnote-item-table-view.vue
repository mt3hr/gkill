<template>
    <div>
        <DnoteItemListView v-for="dnote_items in model_value"
            :key="(dnote_items && dnote_items.length > 0) ? dnote_items[0].id : 0" v-model="dnote_items!"
            :editable="editable" :application_config="application_config" :gkill_api="gkill_api"
            ref="dnote_item_list_views" />
    </div>
</template>

<script lang="ts" setup>
import DnoteItemListView from './dnote-item-list-view.vue';
import type DnoteItemTableProps from './dnote-item-table-props';
import { ref } from 'vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type DnoteItem from '../../classes/dnote/dnote-item';
import AddDnoteItemDialog from '../../pages/dialogs/add-dnote-item-dialog.vue';
import { type KyouViewEmits } from './kyou-view-emits';

const dnote_item_list_views = ref()
const add_dnote_item_dialog = ref<InstanceType<typeof AddDnoteItemDialog> | null>(null);

const model_value = defineModel<Array<Array<DnoteItem>>>()
defineProps<DnoteItemTableProps>()
defineExpose({ load_aggregated_value })
const emits = defineEmits<KyouViewEmits>()

async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
    if (!dnote_item_list_views.value || !model_value.value) {
        return
    }
    for (let i = 0; i < dnote_item_list_views.value.length; i++) {
        if (!(dnote_item_list_views.value as any)[i]) {
            continue
        }
        const dnote_item_list_view = (dnote_item_list_views.value as any)[i]
        await dnote_item_list_view.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded)
    }
}
</script>