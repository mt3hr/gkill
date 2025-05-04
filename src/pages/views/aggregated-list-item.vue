<template>
    <v-card class="pa-0 ma-0 aggregated_list_item"
        @contextmenu.prevent.stop="(e: any) => contextmenu?.show(e, dnote_list_query.id)">
        <div>
            {{ aggregated_item.title }}
        </div>
        <v-row class="pa-0 ma-0">
            <v-col class="pa-0 ma-0" cols="auto">
                {{ aggregated_item.value }}
            </v-col>
        </v-row>
        <KyouListViewDialog v-model="aggregated_item.match_kyous" :kyou_height="180" :width="400"
            :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="''" :is_focused_list="true"
            :closable="false" :highlight_targets="[]" :list_height="list_height" :enable_context_menu="true"
            :enable_dialog="true" :is_readonly_mi_check="true" :show_checkbox="true" :show_footer="true"
            :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true" :show_content_only="false"
            :show_timeis_plaing_end_button="false" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
        <DnoteListQueryContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_list_query="confirm_delete_dnote_list_query_dialog?.show(dnote_list_query)"
            @requested_edit_dnote_list_query="edit_dnote_item_dialog?.show()" ref="contextmenu" />
        <ConfirmDeleteDnoteListQueryDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_list_query="(id) => emits('requested_delete_dnote_list_query', id)"
            ref="confirm_delete_dnote_list_query_dialog" />
        <EditDnoteListDialog :application_config="application_config" :gkill_api="gkill_api"
            :dnote_list_query="dnote_list_query" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_dnote_list_query="(dnote_list_query) => emits('requested_update_dnote_list_query', dnote_list_query)"
            ref="edit_dnote_list_query" />
    </v-card>
</template>
<script lang="ts" setup>
import { computed, ref } from 'vue';
import type AggregatedListItemProps from './aggregated-list-item-props';
import type AggregatedListItemViewEmits from './aggregated-list-item-view-emits';
import KyouListViewDialog from '../dialogs/kyou-list-view-dialog.vue';
import EditDnoteListDialog from '../dialogs/edit-dnote-list-dialog.vue';
import DnoteListQueryContextMenu from './dnote-list-query-context-menu.vue';
import ConfirmDeleteDnoteListQueryDialog from '../dialogs/confirm-delete-dnote-list-query-dialog.vue';

const contextmenu = ref<InstanceType<typeof DnoteListQueryContextMenu> | null>(null);
const confirm_delete_dnote_list_query_dialog = ref<InstanceType<typeof ConfirmDeleteDnoteListQueryDialog> | null>(null);
const edit_dnote_item_dialog = ref<InstanceType<typeof EditDnoteListDialog> | null>(null);

defineProps<AggregatedListItemProps>()
const emits = defineEmits<AggregatedListItemViewEmits>()
const list_height = computed(() => window.screen.height * 7 / 10)
</script>
<style lang="css" scoped>
.aggregated_list_item {
    border-top: 1px solid silver;
}
</style>