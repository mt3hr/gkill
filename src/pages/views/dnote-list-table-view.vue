<template>
    <div>
        <table>
            <tr>
                <td v-for="dnote_list_item in model_value" :key="dnote_list_item.id">
                    <DnoteListView :application_config="application_config" :gkill_api="gkill_api"
                        v-model="dnote_list_item!" :editable="editable"
                        @requested_delete_dnote_list_query="(...id: any[]) => delete_dnote_list_query(id[0] as string)"
                        @requested_update_dnote_list_query="(...dnote_item: any[]) => update_dnote_list_query(dnote_item[0] as DnoteListQuery)"
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
                        @finish_a_aggregate_task="emits('finish_a_aggregate_task')" ref="dnote_list_views" />
                </td>
            </tr>
        </table>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, ref } from 'vue';
import type DnoteListTableViewProps from './dnote-list-table-view-props';
import DnoteListView from './dnote-list-view.vue';
import type { Kyou } from '../../classes/datas/kyou';
import type { Text } from '@/classes/datas/text';
import type { Tag } from '@/classes/datas/tag';
import type { Notification } from '@/classes/datas/notification';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type DnoteListQuery from './dnote-list-query';
import type DnoteListTableViewEmits from './dnote-list-table-view-emits';
import type DnoteItem from '@/classes/dnote/dnote-item';

const dnote_list_views = ref()

defineProps<DnoteListTableViewProps>()
defineExpose({ load_aggregate_grouping_list, reset })
const emits = defineEmits<DnoteListTableViewEmits>()
const model_value = defineModel<Array<DnoteListQuery>>()

async function load_aggregate_grouping_list(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<any> {
    if (!dnote_list_views.value || !model_value.value) {
        return
    }
    const waitPromises = new Array<Promise<any>>()
    for (let i = 0; i < dnote_list_views.value.length; i++) {
        if (!dnote_list_views.value[i]) {
            continue
        }
        waitPromises.push(dnote_list_views.value[i].load_aggregate_grouping_list(abort_controller, kyous, find_kyou_query, kyou_is_loaded))
    }
    return Promise.all(waitPromises)
}


function delete_dnote_list_query(dnote_list_query_id: string): void {
    let delete_target_index: number | null = null;
    for (let i = 0; i < model_value.value!.length; i++) {
        const dnote_list_query = model_value.value![i]
        if (dnote_list_query.id === dnote_list_query_id) {
            delete_target_index = i
            break
        }
    }
    if (delete_target_index === null) {
        return
    }
    model_value.value?.splice(delete_target_index, 1)
}

function update_dnote_list_query(_dnote_list_query: DnoteListQuery): void {
    let delete_target_index: number | null = null;
    for (let i = 0; i < model_value.value!.length; i++) {
        const dnote_list_query = model_value.value![i]
        if (dnote_list_query.id === _dnote_list_query.id) {
            delete_target_index = i
            break
        }
    }
    if (delete_target_index === null) {
        return
    }
    model_value.value?.splice(delete_target_index, 1, _dnote_list_query)
}

async function reset(): Promise<void> {
    if (!dnote_list_views.value || dnote_list_views.value.length === 0) {
        return
    }
    return nextTick(async () => {
        for (let i = 0; i < dnote_list_views.value.length; i++) {
            await dnote_list_views.value[i].reset()
        }
    })
}
</script>