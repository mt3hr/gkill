<template>
    <div @contextmenu.prevent.stop="(e: any) => { if (editable) { contextmenu?.show(e, model_value!.id) } }"
        @dblclick="kyou_list_view_dialog?.show()">
        <table>
            <tr>
                <td>
                    <span class="title">
                        <span>{{ title }}</span>
                        <span>:</span>
                    </span>
                </td>
                <td>
                    <span>{{ prefix }}</span>
                </td>
                <td>
                    <span class="value">
                        <span v-if="!is_lantana_type" :class="value_class">{{ value }}</span>
                        <span v-if="is_lantana_type">
                            <LantanaFlowersView v-if="is_lantana_type" :gkill_api="gkill_api"
                                :application_config="application_config" :mood="mood_value" :editable="false" />
                        </span>
                    </span>
                </td>
                <td>
                    <span>{{ suffix }}</span>
                </td>
            </tr>
        </table>
        <KyouListViewDialog v-model="related_kyous" :kyou_height="180" :width="400"
            :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="''" :is_focused_list="true"
            :closable="false" :highlight_targets="[]" :list_height="list_height" :enable_context_menu="true"
            :enable_dialog="true" :is_readonly_mi_check="true" :show_checkbox="true" :show_footer="false"
            :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true" :show_content_only="false"
            :show_timeis_plaing_end_button="false" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="kyou_list_view_dialog" />
        <DnoteItemListContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_item_list="confirm_delete_dnote_item_list_dialog?.show(model_value!)"
            @requested_edit_dnote_item_list="edit_dnote_item_dialog?.show()" ref="contextmenu" />
        <ConfirmDeleteDnoteItemListDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_list_item="(id) => emits('requested_delete_dnote_item', id)"
            ref="confirm_delete_dnote_item_list_dialog" />
        <EditDnoteItemDialog :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_dnote_item="(dnote_item) => emits('requested_update_dnote_item', dnote_item)"
            ref="edit_dnote_item_dialog" />
    </div>
</template>
<script lang="ts" setup>
import { computed, ref, type Ref } from 'vue'
import type DnoteItemProps from './dnote-item-props'
import type { Kyou } from '../../classes/datas/kyou';
import { DnoteAgregator } from '../../classes/dnote/dnote-aggregator';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import DnoteItemListContextMenu from './dnote-item-list-context-menu.vue';
import ConfirmDeleteDnoteItemListDialog from '../dialogs/confirm-delete-dnote-item-list-dialog.vue';
import EditDnoteItemDialog from '../dialogs/edit-dnote-item-dialog.vue';
import type DnoteItemViewEmits from './dnote-item-view-emits';
import KyouListViewDialog from '../dialogs/kyou-list-view-dialog.vue';
import type DnoteItem from '@/classes/dnote/dnote-item';
import LantanaFlowersView from './lantana-flowers-view.vue';
const contextmenu = ref<InstanceType<typeof DnoteItemListContextMenu> | null>(null);
const confirm_delete_dnote_item_list_dialog = ref<InstanceType<typeof ConfirmDeleteDnoteItemListDialog> | null>(null);
const edit_dnote_item_dialog = ref<InstanceType<typeof EditDnoteItemDialog> | null>(null);
const kyou_list_view_dialog = ref<InstanceType<typeof KyouListViewDialog> | null>(null);

defineProps<DnoteItemProps>()
const emits = defineEmits<DnoteItemViewEmits>()
const model_value = defineModel<DnoteItem>()
defineExpose({ load_aggregated_value })

const title: Ref<string> = ref(model_value.value!.title)
const prefix: Ref<string> = ref(model_value.value!.prefix)
const suffix: Ref<string> = ref(model_value.value!.suffix)
const value: Ref<string> = ref("")
const related_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const list_height = computed(() => window.screen.height * 7 / 10)

const aggregate_target_type = computed(() => model_value.value?.agregate_target?.to_json().type.toString())
const is_lantana_type = computed(() => aggregate_target_type.value.includes("Lantana"))
const is_plus_number_value = computed(() => {
    if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
        if (value.value.startsWith("-")) {
            return false
        } else {
            return true
        }
    }
    return false
})
const is_minus_number_value = computed(() => {
    if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
        if (value.value.toString().startsWith("-")) {
            return true
        }
    }
    return false
})
const value_class = computed(() => {
    if (is_plus_number_value.value) {
        return "plus_value"
    } else if (is_minus_number_value.value) {
        return "minus_value"
    }
    return ""
})
const mood_value = computed(() => Number(value.value).valueOf())
async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
    const dnote_aggregator = new DnoteAgregator(model_value.value!.predicate, model_value.value!.agregate_target)
    const aggregate_result = await dnote_aggregator.agregate(abort_controller, kyous, query, kyou_is_loaded)
    value.value = aggregate_result.result_string
    related_kyous.value = aggregate_result.match_kyous
}

</script>
<style lang="css">
.title {
    font-weight: bold;
}

.value {
    text-align: right;
}
</style>