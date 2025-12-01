<template>
    <div>
        <DnoteItemListView v-for="dnote_items in model_value"
            :key="(dnote_items && dnote_items.length > 0) ? dnote_items[0].id : 0" v-model="dnote_items!"
            :editable="editable" :application_config="application_config" :gkill_api="gkill_api"
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
            @finish_a_aggregate_task="emits('finish_a_aggregate_task')" ref="dnote_item_list_views" />
    </div>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import DnoteItemListView from './dnote-item-list-view.vue';
import type DnoteItemTableProps from './dnote-item-table-props';
import { nextTick, ref } from 'vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type { Text } from '@/classes/datas/text';
import type { Tag } from '@/classes/datas/tag';
import type { Notification } from '@/classes/datas/notification';
import type DnoteItem from '../../classes/dnote/dnote-item';
import type DnoteItemTableViewEmits from './dnote-item-table-view-emits';

const dnote_item_list_views = ref()

const model_value = defineModel<Array<Array<DnoteItem>>>()
defineProps<DnoteItemTableProps>()
defineExpose({ load_aggregated_value, reset })
const emits = defineEmits<DnoteItemTableViewEmits>()

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