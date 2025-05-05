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
                            <span :class="value_class">{{ aggregated_item.value }}</span>
                            <span v-if="is_lantana_type">
                                <LantanaFlowersView :gkill_api="gkill_api" :application_config="application_config"
                                    :mood="mood_value" :editable="false" />
                            </span>
                        </td>
                        <td>
                            <span v-if="!is_lantana_type">{{ dnote_list_query.suffix }}</span>
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>
        <KyouListViewDialog v-model="aggregated_item.match_kyous" :kyou_height="180" :width="400"
            :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="''" :is_focused_list="true"
            :closable="false" :highlight_targets="[]" :list_height="list_height" :enable_context_menu="true"
            :enable_dialog="true" :is_readonly_mi_check="true" :show_checkbox="true" :show_footer="false"
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
import LantanaFlowersView from './lantana-flowers-view.vue';

const kyou_list_view_dialog = ref<InstanceType<typeof KyouListViewDialog> | null>(null);

const props = defineProps<AggregatedListItemProps>()
const emits = defineEmits<AggregatedListItemViewEmits>()
const list_height = computed(() => window.screen.height * 7 / 10)

const aggregate_target_type = computed(() => props.dnote_list_query.aggregate_target.to_json().type.toString())
const is_lantana_type = computed(() => aggregate_target_type.value.includes("Lantana"))
const is_plus_number_value = computed(() => {
    if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
        if (props.aggregated_item.value.toString().startsWith("-")) {
            return false
        } else {
            return true
        }
    }
    return false
})
const is_minus_number_value = computed(() => {
    if (aggregate_target_type.value.includes("Git") || aggregate_target_type.value.includes("Nlog")) {
        if (props.aggregated_item.value.toString().startsWith("-")) {
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
const mood_value = computed(() => Number(props.aggregated_item.value).valueOf())
</script>
<style lang="css" scoped>
.aggregated_list_item {
    border-top: 1px solid silver;
}
</style>
