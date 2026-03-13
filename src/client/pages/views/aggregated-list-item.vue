<template>
    <v-card class="pa-0 ma-0 aggregated_list_item" @dblclick="kyou_list_view_dialog?.show()">
        <div>
            {{ aggregated_item.title }}
        </div>
        <v-row class="pa-0 ma-0">
            <v-col class="pa-0 ma-0" cols="auto">
                <table>
                    <tr>
                        <td>
                            <span>{{ dnote_list_query.prefix }}</span>
                        </td>
                        <td>
                            <span v-if="!is_lantana_type" :class="value_class" v-html="aggregated_item.value"></span>
                            <span v-if="is_lantana_type">
                                <LantanaFlowersView :gkill_api="gkill_api" :application_config="application_config"
                                    :mood="mood_value" :editable="false" />
                            </span>
                        </td>
                        <td>
                            <span>{{ dnote_list_query.suffix }}</span>
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>
        <KyouListViewDialog v-model="aggregated_item.match_kyous" :kyou_height="180" :width="400"
            :application_config="application_config" :gkill_api="gkill_api" :is_focused_list="true"
            :closable="false" :highlight_targets="[]" :list_height="list_height" :enable_context_menu="true"
            :enable_dialog="true" :is_readonly_mi_check="true" :show_checkbox="true" :show_footer="false"
            :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true" :show_content_only="false"
            :show_rep_name="true" :force_show_latest_kyou_info="true" :show_timeis_plaing_end_button="false"
            v-on="crudRelayHandlers"
            ref="kyou_list_view_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { ref } from 'vue'
import type AggregatedListItemProps from './aggregated-list-item-props'
import type AggregatedListItemViewEmits from './aggregated-list-item-view-emits'
import KyouListViewDialog from '../dialogs/kyou-list-view-dialog.vue'
import LantanaFlowersView from './lantana-flowers-view.vue'
import { useAggregatedListItem } from '@/classes/use-aggregated-list-item'

const kyou_list_view_dialog = ref<InstanceType<typeof KyouListViewDialog> | null>(null)

const props = defineProps<AggregatedListItemProps>()
const emits = defineEmits<AggregatedListItemViewEmits>()

const {
    list_height,
    is_lantana_type,
    value_class,
    mood_value,
    crudRelayHandlers,
} = useAggregatedListItem({ props, emits })
</script>
<style lang="css" scoped>
.aggregated_list_item {
    border-top: 1px solid silver;
}
</style>
