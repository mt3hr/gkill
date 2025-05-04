<template>
    <v-card class="pa-0 ma-0 aggregated_list_item" @dblclick="kyou_list_view_dialog?.show()">
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
            @received_messages="(messages) => emits('received_messages', messages)" ref="kyou_list_view_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { computed, ref } from 'vue';
import type AggregatedListItemProps from './aggregated-list-item-props';
import type AggregatedListItemViewEmits from './aggregated-list-item-view-emits';
import KyouListViewDialog from '../dialogs/kyou-list-view-dialog.vue';

const kyou_list_view_dialog = ref<InstanceType<typeof KyouListViewDialog> | null>(null);

defineProps<AggregatedListItemProps>()
const emits = defineEmits<AggregatedListItemViewEmits>()
const list_height = computed(() => window.screen.height * 7 / 10)
</script>
<style lang="css" scoped>
.aggregated_list_item {
    border-top: 1px solid silver;
}
</style>