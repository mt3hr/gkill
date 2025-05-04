<template>
    <div @contextmenu.prevent.stop="(e: any) => contextmenu?.show(e, dnote_item.id)">
        <span class="title">
            <span>{{ title }}</span>
            <span>:</span>
        </span>
        <span class="value">
            <span>{{ prefix }}</span>
            <span>{{ value }}</span>
            <span>{{ suffix }}</span>
        </span>
        <KyouListViewDialog v-model="related_kyous" :kyou_height="180" :width="400"
            :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="''" :is_focused_list="true"
            :closable="false" :highlight_targets="[]" :list_height="list_height" :enable_context_menu="true"
            :enable_dialog="true" :is_readonly_mi_check="true" :show_checkbox="true" :show_footer="true"
            :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true" :show_content_only="false"
            :show_timeis_plaing_end_button="false" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
        <DnoteItemListContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_item_list="confirm_delete_dnote_item_list_dialog?.show(dnote_item)"
            @requested_edit_dnote_item_list="edit_dnote_item_dialog?.show()" ref="contextmenu" />
        <ConfirmDeleteDnoteItemListDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_list_item="(id) => emits('requested_delete_dnote_item', id)"
            ref="confirm_delete_dnote_item_list_dialog" />
        <EditDnoteItemDialog :application_config="application_config" :gkill_api="gkill_api"
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
import { DnoteAggregator } from '../../classes/dnote/dnote-aggregator';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import DnoteItemListContextMenu from './dnote-item-list-context-menu.vue';
import ConfirmDeleteDnoteItemListDialog from '../dialogs/confirm-delete-dnote-item-list-dialog.vue';
import EditDnoteItemDialog from '../dialogs/edit-dnote-item-dialog.vue';
import type DnoteItemViewEmits from './dnote-item-view-emits';
import KyouListViewDialog from '../dialogs/kyou-list-view-dialog.vue';
const contextmenu = ref<InstanceType<typeof DnoteItemListContextMenu> | null>(null);
const confirm_delete_dnote_item_list_dialog = ref<InstanceType<typeof ConfirmDeleteDnoteItemListDialog> | null>(null);
const edit_dnote_item_dialog = ref<InstanceType<typeof EditDnoteItemDialog> | null>(null);

const props = defineProps<DnoteItemProps>()
const emits = defineEmits<DnoteItemViewEmits>()
defineExpose({ load_aggregated_value })

const title: Ref<string> = ref(props.dnote_item.title)
const prefix: Ref<string> = ref(props.dnote_item.prefix)
const suffix: Ref<string> = ref(props.dnote_item.suffix)
const value: Ref<string> = ref("")
const related_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const list_height = computed(() => window.screen.height * 7 / 10)

async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
    const dnote_aggregator = new DnoteAggregator(props.dnote_item.predicate, props.dnote_item.aggregate_target)
    const aggregate_result = await dnote_aggregator.aggregate(abort_controller, kyous, query, kyou_is_loaded)
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