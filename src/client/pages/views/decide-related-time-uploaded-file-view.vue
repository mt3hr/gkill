<template>
    <v-card>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <KyouListView class="decide_related_time_uploaded_files_kyou_list_view"
                    :show_timeis_plaing_end_button="false" :application_config="application_config"
                    :gkill_api="gkill_api" :matched_kyous="uploaded_kyous" :kyou_height="180" :width="400"
                    :list_height="app_content_height.valueOf() - 200" :query="new FindKyouQuery()"
                    :is_focused_list="false" :closable="false"
                    :is_readonly_mi_check="true"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    :is_show_doc_image_toggle_button="false" :is_show_arrow_button="false" :show_rep_name="true"
                    :force_show_latest_kyou_info="true"
                    :scroll_top="0"
                    :show_checkbox="false" :show_footer="false" :show_content_only="false"
                    v-on="kyouListViewHandlers"
                    ref="kyou_list_view" />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <div class="edit_idf_kyou_view_wrap">
                    <EditIDFKyouView v-if="focused_kyou" :application_config="application_config" :gkill_api="gkill_api"
                        :highlight_targets="[]" :kyou="focused_kyou"
                        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                        v-on="editIdfKyouViewHandlers"
                        ref="edit_idf_kyou_view" />
                </div>
            </v-col>
        </v-row>
    </v-card>
</template>
<script setup lang="ts">
import type { DecideRelatedTimeUploadedFileViewProps } from './decide-related-time-uploaded-file-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import EditIDFKyouView from './edit-idf-kyou-view.vue'
import KyouListView from './kyou-list-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { useDecideRelatedTimeUploadedFileView } from '@/classes/use-decide-related-time-uploaded-file-view'

const props = defineProps<DecideRelatedTimeUploadedFileViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // Template refs
    kyou_list_view,
    edit_idf_kyou_view,

    // State
    enable_context_menu,
    enable_dialog,
    focused_kyou,
    kyou_height_px,

    // Event relay objects
    kyouListViewHandlers,
    editIdfKyouViewHandlers,
} = useDecideRelatedTimeUploadedFileView({ props, emits })
</script>
<style lang="css">
.decide_related_time_uploaded_files_kyou_list_view .kyou_in_list .v-card {
    overflow: hidden;
}

.edit_idf_kyou_view_wrap {
    width: calc(400px + 8px);
    max-width: calc(400px + 8px);
    min-width: calc(400px + 8px);
}

.decide_related_time_uploaded_files_kyou_list_view .kyou .v-card {
    overflow: hidden !important;
    height: v-bind(kyou_height_px) !important;
    min-height: v-bind(kyou_height_px) !important;
    max-height: v-bind(kyou_height_px) !important;
    border-top: 1px solid silver;
}

.edit_idf_kyou_view_wrap .kyou {
    overflow: hidden !important;
    width: unset;
    max-width: unset;
    min-width: unset;
    border-top: 1px solid silver;
    height: unset !important;
    min-height: unset !important;
    max-height: unset !important;
}

.edit_idf_kyou_view_wrap .kyou_image {
    width: -webkit-fill-available !important;
    height: fit-content !important;
    max-width: -webkit-fill-available !important;
    max-height: 85vh !important;
}
</style>
