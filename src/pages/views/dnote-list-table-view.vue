<template>
    <div>
        <table>
            <tr>
                <td v-for="dnote_list_item in model_value" :key="dnote_list_item.id">
                    <DnoteListView :application_config="application_config" :gkill_api="gkill_api"
                        v-model="dnote_list_item!" :editable="editable"
                        @requested_delete_dnote_list_query="(id) => delete_dnote_list_query(id)"
                        @requested_update_dnote_list_query="(dnote_item) => update_dnote_list_query(dnote_item)"
                        @deleted_kyou="(kyou) => emits('deleted_kyou', kyou)"
                        @deleted_tag="(tag) => emits('deleted_tag', tag)"
                        @deleted_text="(text) => emits('deleted_text', text)"
                        @deleted_notification="(notification) => emits('deleted_notification', notification)"
                        @registered_kyou="(kyou) => emits('registered_kyou', kyou)"
                        @registered_tag="(tag) => emits('registered_tag', tag)"
                        @registered_text="(text) => emits('registered_text', text)"
                        @registered_notification="(notification) => emits('registered_notification', notification)"
                        @updated_kyou="(kyou) => emits('updated_kyou', kyou)"
                        @updated_tag="(tag) => emits('updated_tag', tag)"
                        @updated_text="(text) => emits('updated_text', text)"
                        @updated_notification="(notification) => emits('updated_notification', notification)"
                        ref="dnote_list_views" />
                </td>
            </tr>
        </table>
    </div>
</template>
<script lang="ts" setup>
import { nextTick, ref } from 'vue';
import type DnoteListTableViewProps from './dnote-list-table-view-props';
import DnoteListView from './dnote-list-view.vue';
import type { Kyou } from '../../classes/datas/kyou';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type DnoteListQuery from './dnote-list-query';
import { type KyouViewEmits } from './kyou-view-emits';

const dnote_list_views = ref()

defineProps<DnoteListTableViewProps>()
defineExpose({ load_aggregate_grouping_list, reset })
const emits = defineEmits<KyouViewEmits>()
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