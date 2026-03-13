<template>
    <v-card class="pa-0 ma-0 related_kyou_list_item" :draggable="editable" :class="{ draggable: editable }"
        @dragstart="drag_start" @dragover="dragover" @drop="drop"
        @contextmenu.prevent.stop="(e: any) => { if (editable) { show_context_menu(e) } }"
        @dblclick="() => { if (editable) { show_edit_ryuu_item_dialog() } }">
        <table>
            <tr>
                <td>
                    <span>
                        {{ model_value?.title }}
                    </span>
                </td>
                <td>
                    <span>:</span>
                </td>
                <td>
                    <v-row class="pa-0 ma-0">
                        <v-col class="pa-0 ma-0" cols="auto">
                            <table>
                                <tr>
                                    <td>
                                        <span>{{ model_value?.prefix }}</span>
                                    </td>

                                    <td v-if="is_no_data">
                                        ---
                                    </td>

                                    <td v-if="match_kyou && !is_no_data">
                                        <span
                                            v-if="(match_kyou.data_type.startsWith('lantana') && match_kyou.typed_lantana)">
                                            <LantanaFlowersView :gkill_api="gkill_api"
                                                :application_config="application_config"
                                                :mood="match_kyou.typed_lantana.mood" :editable="false"
                                                @dblclick="() => { if (editable) { show_edit_ryuu_item_dialog() } else { show_kyou_dialog() } }" />
                                        </span>

                                        <span v-if="(match_kyou.data_type.startsWith('kc') && match_kyou.typed_kc)"
                                            @dblclick="() => { if (editable) { show_edit_ryuu_item_dialog() } else { show_kyou_dialog() } }">
                                            {{ match_kyou.typed_kc.num_value }}
                                        </span>

                                        <KyouView :is_image_request_to_thumb_size="false"
                                            v-if="!(match_kyou.data_type.startsWith('lantana') && match_kyou.typed_lantana) && !(match_kyou.data_type.startsWith('kc') && match_kyou.typed_kc)"
                                            :application_config="application_config" :gkill_api="gkill_api"
                                            :highlight_targets="[]" :is_image_view="false" :kyou="match_kyou"
                                            :show_checkbox="false" :show_content_only="true"
                                            :show_mi_create_time="false" :show_mi_estimate_end_time="false"
                                            :show_mi_estimate_start_time="false" :show_mi_limit_time="false"
                                            :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="false"
                                            :height="'fit-content'" :enable_context_menu="enable_context_menu"
                                            :enable_dialog="enable_dialog" :show_attached_timeis="false"
                                            :show_update_time="false" :show_related_time="false" :width="'fit-content'"
                                            :is_readonly_mi_check="true" :show_rep_name="false"
                                            :force_show_latest_kyou_info="true" :show_attached_tags="true"
                                            :show_attached_texts="true" :show_attached_notifications="true"
                                            v-on="kyouViewRelayHandlers" />
                                    </td>

                                    <td>
                                        <span>{{ model_value?.suffix }}</span>
                                    </td>
                                </tr>
                            </table>
                        </v-col>
                    </v-row>
                </td>
            </tr>
        </table>

        <KyouDialog v-if="match_kyou" :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[]" :kyou="match_kyou" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" :is_readonly_mi_check="false" :show_timeis_plaing_end_button="true"
            v-on="kyouDialogRelayHandlers"
            ref="kyou_dialog" />

        <RyuuListItemContextMenu :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
            v-on="contextMenuRelayHandlers"
            ref="contextmenu" />

        <EditRyuuItemDialog :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
            ref="edit_related_kyou_query_dialog" />
    </v-card>
</template>

<script lang="ts" setup>
import EditRyuuItemDialog from '../dialogs/edit-ryuu-item-dialog.vue'
import KyouView from './kyou-view.vue'
import type RyuuListItemViewEmits from './ryuu-list-item-view-emits'
import type RyuuListItemViewProps from './ryuu-list-item-view-props'
import KyouDialog from '../dialogs/kyou-dialog.vue'
import LantanaFlowersView from './lantana-flowers-view.vue'
import type RelatedKyouQuery from '../../classes/dnote/related-kyou-query'
import RyuuListItemContextMenu from './ryuu-list-item-context-menu.vue'
import { useRyuuListItemView } from '@/classes/use-ryuu-list-item-view'

const model_value = defineModel<RelatedKyouQuery>()
const props = defineProps<RyuuListItemViewProps>()
const emits = defineEmits<RyuuListItemViewEmits>()

const {
    // Template refs
    kyou_dialog,
    contextmenu,
    edit_related_kyou_query_dialog,

    // State
    match_kyou,
    is_no_data,

    // Methods
    drag_start,
    dragover,
    drop,
    load_related_kyou,
    show_kyou_dialog,
    show_context_menu,
    show_edit_ryuu_item_dialog,

    // Event relay objects
    kyouViewRelayHandlers,
    kyouDialogRelayHandlers,
    contextMenuRelayHandlers,
} = useRyuuListItemView({ props, emits, model_value })

defineExpose({ load_related_kyou })
</script>

<style lang="css" scoped>
.related_kyou_list_item {
    border-top: 1px solid silver;
}

.related_kyou_list_item.draggable {
    cursor: grab;
}

.related_kyou_list_item.draggable:active {
    cursor: grabbing;
}
</style>
