<template>
    <div @contextmenu.prevent.stop="(e: any) => { if (editable) { contextmenu?.show(e, model_value!.id) } }"
        @dblclick="kyou_list_view_dialog?.show()">
        <table>
            <tr>
                <td>
                    <span class="title">
                        <span>{{ model_value!.title }}</span>
                        <span>:</span>
                    </span>
                </td>
                <td>
                    <span>{{ model_value!.prefix }}</span>
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
                    <span>{{ model_value!.suffix }}</span>
                </td>
            </tr>
        </table>
        <KyouListViewDialog v-model="related_kyous" :kyou_height="180" :width="400"
            :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="''" :is_focused_list="true"
            :closable="false" :highlight_targets="[]" :list_height="list_height" :enable_context_menu="true"
            :enable_dialog="true" :is_readonly_mi_check="true" :show_checkbox="true" :show_footer="false"
            :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true" :show_content_only="false"
            :show_rep_name="true" :force_show_latest_kyou_info="true" :show_timeis_plaing_end_button="false"
            @deleted_kyou="(...kyou: any[]) => emits('deleted_kyou', kyou[0] as Kyou)" @deleted_tag="(...tag: any[]) => emits('deleted_tag', tag[0] as Tag)"
            @deleted_text="(...text :any[]) => emits('deleted_text', text[0] as Text)"
            @deleted_notification="(...notification :any[]) => emits('deleted_notification', notification[0] as Notification)"
            @registered_kyou="(...kyou: any[]) => emits('registered_kyou', kyou[0] as Kyou)"
            @registered_tag="(...tag: any[]) => emits('registered_tag', tag[0] as Tag)"
            @registered_text="(...text: any[]) => emits('registered_text', text[0] as Text)"
            @registered_notification="(...notification: any[]) => emits('registered_notification', notification[0] as Notification)"
            @updated_kyou="(...kyou: any[]) => emits('updated_kyou', kyou[0] as Kyou)" @updated_tag="(...tag :any[]) => emits('updated_tag', tag[0] as Tag)"
            @updated_text="(...text: any[]) => emits('updated_text', text[0] as Text)"
            @updated_notification="(...notification :any[]) => emits('updated_notification', notification[0] as Notification)"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" ref="kyou_list_view_dialog" />
        <DnoteItemListContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_delete_dnote_item_list="confirm_delete_dnote_item_list_dialog?.show(model_value!)"
            @requested_edit_dnote_item_list="edit_dnote_item_dialog?.show()" ref="contextmenu" />
        <ConfirmDeleteDnoteItemListDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_delete_dnote_list_item="(...id :any[]) => emits('requested_delete_dnote_item', id[0] as string)"
            ref="confirm_delete_dnote_item_list_dialog" />
        <EditDnoteItemDialog :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_dnote_item="(...dnote_item: any[]) => emits('requested_update_dnote_item', dnote_item[0] as DnoteItem)"
            ref="edit_dnote_item_dialog" />
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, ref, watch, type Ref } from 'vue'
import type DnoteItemProps from './dnote-item-props'
import { DnoteAgregator } from '../../classes/dnote/dnote-aggregator';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import DnoteItemListContextMenu from './dnote-item-list-context-menu.vue';
import ConfirmDeleteDnoteItemListDialog from '../dialogs/confirm-delete-dnote-item-list-dialog.vue';
import EditDnoteItemDialog from '../dialogs/edit-dnote-item-dialog.vue';
import type DnoteItemViewEmits from './dnote-item-view-emits';
import KyouListViewDialog from '../dialogs/kyou-list-view-dialog.vue';
import type DnoteItem from '@/classes/dnote/dnote-item';
import LantanaFlowersView from './lantana-flowers-view.vue';
import type { Kyou } from '../../classes/datas/kyou';
import type { Text } from '@/classes/datas/text';
import type { Tag } from '@/classes/datas/tag';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
const contextmenu = ref<InstanceType<typeof DnoteItemListContextMenu> | null>(null);
const confirm_delete_dnote_item_list_dialog = ref<InstanceType<typeof ConfirmDeleteDnoteItemListDialog> | null>(null);
const edit_dnote_item_dialog = ref<InstanceType<typeof EditDnoteItemDialog> | null>(null);
const kyou_list_view_dialog = ref<InstanceType<typeof KyouListViewDialog> | null>(null);

defineProps<DnoteItemProps>()
const emits = defineEmits<DnoteItemViewEmits>()
const model_value = defineModel<DnoteItem>()
defineExpose({ load_aggregated_value, reset })

const value = ref("")
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
async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean): Promise<any> {
    related_kyous.value.splice(0)
    const dnote_aggregator = new DnoteAgregator(model_value.value!.predicate, model_value.value!.agregate_target)
    const aggregate_result = await dnote_aggregator.agregate(abort_controller, kyous, query, kyou_is_loaded)
    value.value = aggregate_result.result_string.replace("<br>", "")
    related_kyous.value.splice(0, Infinity, ...aggregate_result.match_kyous)
    emits('finish_a_aggregate_task')
}

async function reset(): Promise<void> {
    value.value = ""
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