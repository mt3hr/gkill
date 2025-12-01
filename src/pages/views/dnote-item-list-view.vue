<template>
    <div>
        <DnoteItemView v-for="dnote_item in model_value" v-model="dnote_item!" :key="dnote_item.id" :editable="editable"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_delete_dnote_item="(...id: any[]) => delete_dnote_item(id[0] as string)"
            @requested_update_dnote_item="(...dnote_item: any[]) => update_dnote_item(dnote_item[0] as DnoteItem)"
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
            @finish_a_aggregate_task="emits('finish_a_aggregate_task')" ref="dnote_item_views" />
    </div>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, ref } from 'vue';
import type DnoteItemListViewProps from './dnote-item-list-view-props';
import DnoteItemView from './dnote-item-view.vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type { Text } from '@/classes/datas/text';
import type { Tag } from '@/classes/datas/tag';
import type { Notification } from '@/classes/datas/notification';
import type DnoteItem from '../../classes/dnote/dnote-item';
import type DnoteItemListViewEmits from './dnote-item-list-view-emits';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

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